package dmarcrpt

// Initially generated by xsdgen, then modified.

// Feedback is the top-level XML field returned.
type Feedback struct {
	Version         string          `xml:"version"`
	ReportMetadata  ReportMetadata  `xml:"report_metadata"`
	PolicyPublished PolicyPublished `xml:"policy_published"`
	Records         []ReportRecord  `xml:"record"`
}

type ReportMetadata struct {
	OrgName          string    `xml:"org_name"`
	Email            string    `xml:"email"`
	ExtraContactInfo string    `xml:"extra_contact_info,omitempty"`
	ReportID         string    `xml:"report_id"`
	DateRange        DateRange `xml:"date_range"`
	Errors           []string  `xml:"error,omitempty"`
}

type DateRange struct {
	Begin int64 `xml:"begin"`
	End   int64 `xml:"end"`
}

// PolicyPublished is the policy as found in DNS for the domain.
type PolicyPublished struct {
	Domain           string      `xml:"domain"`
	ADKIM            Alignment   `xml:"adkim,omitempty"`
	ASPF             Alignment   `xml:"aspf,omitempty"`
	Policy           Disposition `xml:"p"`
	SubdomainPolicy  Disposition `xml:"sp"`
	Percentage       int         `xml:"pct"`
	ReportingOptions string      `xml:"fo"`
}

// Alignment is the identifier alignment.
type Alignment string

const (
	AlignmentRelaxed Alignment = "r" // Subdomains match the DMARC from-domain.
	AlignmentStrict  Alignment = "s" // Only exact from-domain match.
)

// Disposition is the requested action for a DMARC fail as specified in the
// DMARC policy in DNS.
type Disposition string

const (
	DispositionNone       Disposition = "none"
	DispositionQuarantine Disposition = "quarantine"
	DispositionReject     Disposition = "reject"
)

type ReportRecord struct {
	Row         Row         `xml:"row"`
	Identifiers Identifiers `xml:"identifiers"`
	AuthResults AuthResults `xml:"auth_results"`
}

type Row struct {
	// SourceIP must match the pattern ((1?[0-9]?[0-9]|2[0-4][0-9]|25[0-5]).){3}
	// (1?[0-9]?[0-9]|2[0-4][0-9]|25[0-5])|
	// ([A-Fa-f0-9]{1,4}:){7}[A-Fa-f0-9]{1,4}
	SourceIP        string          `xml:"source_ip"`
	Count           int             `xml:"count"`
	PolicyEvaluated PolicyEvaluated `xml:"policy_evaluated"`
}

type PolicyEvaluated struct {
	Disposition Disposition            `xml:"disposition"`
	DKIM        DMARCResult            `xml:"dkim"`
	SPF         DMARCResult            `xml:"spf"`
	Reasons     []PolicyOverrideReason `xml:"reason,omitempty"`
}

// DMARCResult is the final validation and alignment verdict for SPF and DKIM.
type DMARCResult string

const (
	DMARCPass DMARCResult = "pass"
	DMARCFail DMARCResult = "fail"
)

type PolicyOverrideReason struct {
	Type    PolicyOverride `xml:"type"`
	Comment string         `xml:"comment,omitempty"`
}

// PolicyOverride is a reason the requested DMARC policy from the DNS record
// was not applied.
type PolicyOverride string

const (
	PolicyOverrideForwarded        PolicyOverride = "forwarded"
	PolicyOverrideSampledOut       PolicyOverride = "sampled_out"
	PolicyOverrideTrustedForwarder PolicyOverride = "trusted_forwarder"
	PolicyOverrideMailingList      PolicyOverride = "mailing_list"
	PolicyOverrideLocalPolicy      PolicyOverride = "local_policy"
	PolicyOverrideOther            PolicyOverride = "other"
)

type Identifiers struct {
	EnvelopeTo   string `xml:"envelope_to,omitempty"`
	EnvelopeFrom string `xml:"envelope_from"`
	HeaderFrom   string `xml:"header_from"`
}

type AuthResults struct {
	DKIM []DKIMAuthResult `xml:"dkim,omitempty"`
	SPF  []SPFAuthResult  `xml:"spf"`
}

type DKIMAuthResult struct {
	Domain      string     `xml:"domain"`
	Selector    string     `xml:"selector,omitempty"`
	Result      DKIMResult `xml:"result"`
	HumanResult string     `xml:"human_result,omitempty"`
}

type DKIMResult string

const (
	DKIMNone      DKIMResult = "none"
	DKIMPass      DKIMResult = "pass"
	DKIMFail      DKIMResult = "fail"
	DKIMPolicy    DKIMResult = "policy"
	DKIMNeutral   DKIMResult = "neutral"
	DKIMTemperror DKIMResult = "temperror"
	DKIMPermerror DKIMResult = "permerror"
)

type SPFAuthResult struct {
	Domain string         `xml:"domain"`
	Scope  SPFDomainScope `xml:"scope"`
	Result SPFResult      `xml:"result"`
}

type SPFDomainScope string

const (
	SPFDomainScopeHelo     SPFDomainScope = "helo"  // SMTP EHLO
	SPFDomainScopeMailFrom SPFDomainScope = "mfrom" // SMTP "MAIL FROM".
)

type SPFResult string

const (
	SPFNone      SPFResult = "none"
	SPFNeutral   SPFResult = "neutral"
	SPFPass      SPFResult = "pass"
	SPFFail      SPFResult = "fail"
	SPFSoftfail  SPFResult = "softfail"
	SPFTemperror SPFResult = "temperror"
	SPFPermerror SPFResult = "permerror"
)