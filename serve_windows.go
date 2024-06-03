package main

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"github.com/mjl-/mox/config"
	"github.com/mjl-/mox/mox-"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/mjl-/mox/dns"
	"github.com/mjl-/mox/dnsbl"
	"github.com/mjl-/mox/message"
	"github.com/mjl-/mox/metrics"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/moxvar"
	"github.com/mjl-/mox/queue"
	"github.com/mjl-/mox/store"
	"github.com/mjl-/mox/updates"
)

var metricDNSBL = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "mox_dnsbl_ips_success",
		Help: "DNSBL lookups to configured DNSBLs of our IPs.",
	},
	[]string{
		"zone",
		"ip",
	},
)

type key struct {
	zone dns.Domain
	ip   string
}

func cmdServe(c *cmd) {
	c.help = `Start mox, serving SMTP/IMAP/HTTPS.

Incoming email is accepted over SMTP. Email can be retrieved by users using
IMAP. HTTP listeners are started for the admin/account web interfaces, and for
automated TLS configuration. Missing essential TLS certificates are immediately
requested, other TLS certificates are requested on demand.
`
	args := c.Parse()
	if len(args) != 0 {
		c.Usage()
	}

	setupLogging()

	log := c.log

	loadConfiguration(log)

	startChildProcess(log)

	initializeSMTPTransactionIDs(log)

	startServer(log)

	if mox.Conf.Static.CheckUpdates {
		go checkForUpdatesPeriodically(log)
	}

	go monitorDNSBL(log)

	startControlListener(log)

	removeOldTemporaryFiles(log)

	waitForShutdownSignal(log)
}

func setupLogging() {
	mlog.Logfmt = true
	mox.Conf.Log[""] = mlog.LevelDebug
	mlog.SetConfig(mox.Conf.Log)
}

func loadConfiguration(log mlog.Log) {
	mox.MustLoadConfig(true, true)
	log.Print("starting",
		slog.String("version", moxvar.Version),
		slog.Any("pid", os.Getpid()))
}

func startChildProcess(log mlog.Log) {
	executable, err := os.Executable()
	if err != nil {
		log.Fatalx("get executable path", err)
	}

	var procInfo syscall.ProcessInformation
	startupInfo := &syscall.StartupInfo{}
	args := []string{executable, "serve", "--child"}
	err = syscall.CreateProcess(
		nil,
		syscall.StringToUTF16Ptr(executable+" "+strings.Join(args, " ")),
		nil,
		nil,
		true,
		0,
		nil,
		nil,
		startupInfo,
		&procInfo,
	)
	if err != nil {
		log.Fatalx("create child process", err)
	}
	defer syscall.CloseHandle(procInfo.Process)
	defer syscall.CloseHandle(procInfo.Thread)

	_, err = syscall.WaitForSingleObject(procInfo.Process, syscall.INFINITE)
	if err != nil {
		log.Fatalx("wait for child process", err)
	}

	var exitCode uint32
	err = syscall.GetExitCodeProcess(procInfo.Process, &exitCode)
	if err != nil {
		log.Fatalx("get child process exit code", err)
	}

	log.Print("child process exited", slog.Any("exitcode", exitCode))
}

func initializeSMTPTransactionIDs(log mlog.Log) {
	recvidpath := mox.DataDirPath("receivedid.key")
	recvidbuf, err := os.ReadFile(recvidpath)
	if err != nil || len(recvidbuf) != 16+8 {
		recvidbuf = make([]byte, 16+8)
		if _, err := cryptorand.Read(recvidbuf); err != nil {
			log.Fatalx("reading random recvid data", err)
		}
		if err := os.WriteFile(recvidpath, recvidbuf, 0660); err != nil {
			log.Fatalx("writing recvidpath", err, slog.String("path", recvidpath))
		}
	}
	if err := mox.ReceivedIDInit(recvidbuf[:16], recvidbuf[16:]); err != nil {
		log.Fatalx("init receivedid", err)
	}
}

func startServer(log mlog.Log) {
	const mtastsdbRefresher = true
	const skipForkExec = true
	if err := start(mtastsdbRefresher, !mox.Conf.Static.NoOutgoingDMARCReports, !mox.Conf.Static.NoOutgoingTLSReports, skipForkExec); err != nil {
		log.Fatalx("start", err)
	}
	log.Print("ready to serve")
}

func checkForUpdatesPeriodically(log mlog.Log) {
	for {
		next := checkUpdates(log)
		time.Sleep(next)
	}
}

func checkUpdates(log mlog.Log) time.Duration {
	next := 24 * time.Hour
	current, lastknown, mtime, err := mox.LastKnown()
	if err != nil {
		log.Infox("determining own version before checking for updates, trying again in 24h", err)
		return next
	}

	if !mtime.IsZero() && time.Since(mtime) < 24*time.Hour {
		d := 24*time.Hour - time.Since(mtime)
		log.Debug("sleeping for next check for updates", slog.Duration("sleep", d))
		time.Sleep(d)
		next = 0
	}
	now := time.Now()
	if err := os.Chtimes(mox.DataDirPath("lastknownversion"), now, now); err != nil {
		if !os.IsNotExist(err) {
			log.Infox("setting mtime on lastknownversion file, continuing", err)
		}
	}

	log.Debug("checking for updates", slog.Any("lastknown", lastknown))
	updatesctx, updatescancel := context.WithTimeout(mox.Context, time.Minute)
	latest, _, changelog, err := updates.Check(updatesctx, log.Logger, dns.StrictResolver{Log: log.Logger}, dns.Domain{ASCII: changelogDomain}, lastknown, changelogURL, changelogPubKey)
	updatescancel()
	if err != nil {
		log.Infox("checking for updates", err, slog.Any("latest", latest))
		return next
	}
	if !latest.After(lastknown) {
		log.Debug("no new version available")
		return next
	}
	if len(changelog.Changes) == 0 {
		log.Info("new version available, but changelog is empty, ignoring", slog.Any("latest", latest))
		return next
	}

	sendUpdateNotification(log, current, lastknown, latest, *changelog)

	return next
}

func sendUpdateNotification(log mlog.Log, current, lastknown, latest updates.Version, changelog updates.Changelog) {
	var cl string
	for _, c := range changelog.Changes {
		cl += "----\n\n" + strings.TrimSpace(c.Text) + "\n\n"
	}
	cl += "----"

	a, err := store.OpenAccount(log, mox.Conf.Static.Postmaster.Account)
	if err != nil {
		log.Infox("open account for postmaster changelog delivery", err)
		return
	}
	defer func() {
		err := a.Close()
		log.Check(err, "closing account")
	}()
	f, err := store.CreateMessageTemp(log, "changelog")
	if err != nil {
		log.Infox("making temporary message file for changelog delivery", err)
		return
	}
	defer store.CloseRemoveTempFile(log, f, "message for changelog delivery")

	m := store.Message{
		Received: time.Now(),
		Flags:    store.Flags{Flagged: true},
	}
	n, err := fmt.Fprintf(f, "Date: %s\r\nSubject: mox %s available\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8-bit\r\n\r\nHi!\r\n\r\nVersion %s of mox is available, this install is at %s.\r\n\r\nChanges:\r\n\r\n%s\r\n\r\nRemember to make a backup with \"mox backup\" before upgrading.\r\nPlease report any issues at https://github.com/mjl-/mox, thanks!\r\n\r\nCheers,\r\nmox\r\n", time.Now().Format(message.RFC5322Z), latest, latest, current, strings.ReplaceAll(cl, "\n", "\r\n"))
	if err != nil {
		log.Infox("writing temporary message file for changelog delivery", err)
		return
	}
	m.Size = int64(n)

	var derr error
	a.WithWLock(func() {
		derr = a.DeliverMailbox(log, mox.Conf.Static.Postmaster.Mailbox, &m, f)
	})
	if derr != nil {
		log.Errorx("changelog delivery", derr)
		return
	}

	log.Info("delivered changelog",
		slog.Any("current", current),
		slog.Any("lastknown", lastknown),
		slog.Any("latest", latest))
	if err := mox.StoreLastKnown(latest); err != nil {
		log.Infox("updating last known version", err)
	}
}

func startControlListener(log mlog.Log) {
	ctl, err := net.Listen("tcp", "localhost:12345")
	if err != nil {
		log.Fatalx("listen on ctl socket", err)
	}
	go func() {
		for {
			conn, err := ctl.Accept()
			if err != nil {
				log.Printx("accept for ctl", err)
				continue
			}
			cid := mox.Cid()
			ctx := context.WithValue(mox.Context, mlog.CidKey, cid)
			go servectl(ctx, log.WithCid(cid), conn, func() { shutdown(log) })
		}
	}()
}

func removeOldTemporaryFiles(log mlog.Log) {
	tmpdir := mox.DataDirPath("tmp")
	os.MkdirAll(tmpdir, 0770)
	tmps, err := os.ReadDir(tmpdir)
	if err != nil {
		log.Errorx("listing files in tmpdir", err)
	} else {
		now := time.Now()
		for _, e := range tmps {
			if fi, err := e.Info(); err != nil {
				log.Errorx("stat tmp file", err, slog.String("filename", e.Name()))
			} else if now.Sub(fi.ModTime()) > 7*24*time.Hour && !fi.IsDir() {
				p := filepath.Join(tmpdir, e.Name())
				if err := os.Remove(p); err != nil {
					log.Errorx("removing stale temporary file", err, slog.String("path", p))
				} else {
					log.Info("removed stale temporary file", slog.String("path", p))
				}
			}
		}
	}
}

func waitForShutdownSignal(log mlog.Log) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	sig := <-sigc
	log.Print("shutting down, waiting max 3s for existing connections", slog.Any("signal", sig))
	shutdown(log)
	os.Exit(0)
}

func monitorDNSBL(log mlog.Log) {
	defer func() {
		x := recover()
		if x != nil {
			log.Error("monitordnsbl panic", slog.Any("panic", x))
			debug.PrintStack()
			metrics.PanicInc(metrics.Serve)
		}
	}()

	publicListener := mox.Conf.Static.Listeners["public"]

	prevResults := map[key]struct{}{}

	var last time.Time
	var lastConns int64

	resolver := dns.StrictResolver{Pkg: "dnsblmonitor"}
	var sleep time.Duration
	for {
		time.Sleep(sleep)
		conns := queue.ConnectionCounter()
		if sleep > 0 && conns < lastConns+100 && time.Since(last) < 3*time.Hour {
			continue
		}
		sleep = 5 * time.Minute
		lastConns = conns
		last = time.Now()

		zones := gatherZones(publicListener, log)
		_, publicIPs, publicIPstrs := gatherPublicIPs(log)
		updateMetrics(log, resolver, zones, publicIPs, publicIPstrs, prevResults)
	}
}

func gatherZones(publicListener config.Listener, log mlog.Log) []dns.Domain {
	zones := append([]dns.Domain{}, publicListener.SMTP.DNSBLZones...)
	conf := mox.Conf.DynamicConfig()
	for _, zone := range conf.MonitorDNSBLZones {
		if !slices.Contains(zones, zone) {
			zones = append(zones, zone)
		}
	}
	return zones
}

func gatherPublicIPs(log mlog.Log) ([]net.IP, []net.IP, []string) {
	ips, err := mox.IPs(mox.Context, false)
	if err != nil {
		log.Errorx("listing ips for dnsbl monitor", err)
		return nil, nil, nil
	}
	var publicIPs []net.IP
	var publicIPstrs []string
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() {
			continue
		}
		publicIPs = append(publicIPs, ip)
		publicIPstrs = append(publicIPstrs, ip.String())
	}
	return ips, publicIPs, publicIPstrs
}

func updateMetrics(log mlog.Log, resolver dns.StrictResolver, zones []dns.Domain, publicIPs []net.IP, publicIPstrs []string, prevResults map[key]struct{}) {
	for k := range prevResults {
		if !slices.Contains(zones, k.zone) || !slices.Contains(publicIPstrs, k.ip) {
			metricDNSBL.DeleteLabelValues(k.zone.Name(), k.ip)
			delete(prevResults, k)
		}
	}

	for _, ip := range publicIPs {
		for _, zone := range zones {
			status, expl, err := dnsbl.Lookup(mox.Context, log.Logger, resolver, zone, ip)
			if err != nil {
				log.Errorx("dnsbl monitor lookup", err,
					slog.Any("ip", ip),
					slog.Any("zone", zone),
					slog.String("expl", expl),
					slog.Any("status", status))
			}
			var v float64
			if status == dnsbl.StatusPass {
				v = 1
			}
			metricDNSBL.WithLabelValues(zone.Name(), ip.String()).Set(v)
			k := key{zone, ip.String()}
			prevResults[k] = struct{}{}

			time.Sleep(time.Second)
		}
	}
}
