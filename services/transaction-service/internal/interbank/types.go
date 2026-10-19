package interbank

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Config struct {
	Bank, Partner, PartnerURL, RabbitURL string
	PartnerKey                           string
}

func Load(rabbit string) (Config, error) {
	c := Config{Bank: os.Getenv("INTERBANK_BANK_CODE"), Partner: os.Getenv("INTERBANK_PARTNER_BANK_CODE"), PartnerURL: strings.TrimRight(os.Getenv("INTERBANK_PARTNER_BASE_URL"), "/"), PartnerKey: os.Getenv("INTERBANK_PARTNER_API_KEY"), RabbitURL: rabbit}
	if c.Bank == "" {
		c.Bank = "BANK03"
	}
	if c.Partner == "" {
		c.Partner = "BANK14"
	}
	if !bankPattern.MatchString(c.Bank) || !bankPattern.MatchString(c.Partner) || c.Bank == c.Partner {
		return c, errors.New("invalid interbank bank codes")
	}
	u, e := url.Parse(c.PartnerURL)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/interbank/v1" || strings.HasSuffix(u.Hostname(), "example.com") || len(c.PartnerKey) < 16 {
		return c, errors.New("interbank partner HTTPS URL and secret required")
	}
	return c, nil
}

var bankPattern = regexp.MustCompile(`^BANK[0-9]{2}$`)
var accountPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,34}$`)
var amountPattern = regexp.MustCompile(`^[0-9]{1,5}\.[0-9]{2}$`)

type Transfer struct {
	ID                 string    `json:"interbankTransferId"`
	Origin             string    `json:"originBankCode"`
	Destination        string    `json:"destinationBankCode"`
	OriginAccount      string    `json:"originAccount"`
	DestinationAccount string    `json:"destinationAccount"`
	Amount             string    `json:"amount"`
	Currency           string    `json:"currency"`
	Description        string    `json:"description,omitempty"`
	Created            time.Time `json:"createdAt"`
	Expires            time.Time `json:"expiresAt"`
}

func Cents(amount string) (int64, error) {
	if !amountPattern.MatchString(amount) {
		return 0, errors.New("invalid amount")
	}
	n, e := strconv.ParseInt(strings.ReplaceAll(amount, ".", ""), 10, 64)
	return n, e
}
func (t Transfer) Valid() bool {
	id, e := uuid.Parse(t.ID)
	_, a := Cents(t.Amount)
	return e == nil && id != uuid.Nil && len(t.ID) == 36 && bankPattern.MatchString(t.Origin) && bankPattern.MatchString(t.Destination) && t.Origin != t.Destination && accountPattern.MatchString(t.OriginAccount) && accountPattern.MatchString(t.DestinationAccount) && a == nil && t.Currency == "GTQ" && utf8.RuneCountInString(t.Description) <= 140 && !t.Created.IsZero() && t.Created.Location() != nil && t.Expires.Sub(t.Created) == 180*time.Second
}
func strict(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

type Result struct {
	ID          string    `json:"interbankTransferId"`
	Destination string    `json:"destinationBankCode"`
	Status      string    `json:"status"`
	Reason      string    `json:"reasonCode,omitempty"`
	Message     string    `json:"reasonMessage,omitempty"`
	Processed   time.Time `json:"processedAt"`
}

func (r Result) Valid() bool {
	if id, err := uuid.Parse(r.ID); err != nil || id == uuid.Nil || len(r.ID) != 36 || !bankPattern.MatchString(r.Destination) {
		return false
	}
	if r.Status != "ACCEPTED" && r.Status != "REJECTED" {
		return false
	}
	if r.Reason != "" {
		switch r.Reason {
		case "ACCOUNT_NOT_FOUND", "ACCOUNT_INACTIVE", "AMOUNT_LIMIT_EXCEEDED", "CURRENCY_NOT_SUPPORTED", "EXPIRED", "CANCELLED", "PROCESSING_ERROR":
		default:
			return false
		}
	}
	return !r.Processed.IsZero() && utf8.RuneCountInString(r.Message) <= 200
}

// Request is an internal RPC envelope. Peer/customer identity is assigned only
// by the Gateway after API-key/JWT validation, never from an external JSON body.
type Request struct {
	Action   string          `json:"action"`
	ID       string          `json:"id,omitempty"`
	Key      string          `json:"key,omitempty"`
	Customer string          `json:"customer,omitempty"`
	Peer     string          `json:"peer,omitempty"`
	Body     json.RawMessage `json:"body,omitempty"`
	Date     string          `json:"date,omitempty"`
	Page     int             `json:"page,omitempty"`
	PageSize int             `json:"pageSize,omitempty"`
}
type AccountCommand struct {
	Original string    `json:"original,omitempty"`
	ID       string    `json:"id"`
	Action   string    `json:"action"`
	Account  string    `json:"account"`
	Customer string    `json:"customer"`
	Cents    int64     `json:"cents"`
	Expires  time.Time `json:"expires"`
}

type Cancel struct {
	Origin string `json:"originBankCode"`
	Reason string `json:"reason"`
}
type Reversal struct {
	ID       string `json:"reversalId"`
	Original string `json:"originalTransferId"`
	Origin   string `json:"originBankCode"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (r Reversal) Valid() bool {
	a, e := uuid.Parse(r.ID)
	b, f := uuid.Parse(r.Original)
	n, g := Cents(r.Amount)
	return e == nil && f == nil && a != uuid.Nil && b != uuid.Nil && len(r.ID) == 36 && len(r.Original) == 36 && bankPattern.MatchString(r.Origin) && r.Currency == "GTQ" && g == nil && n >= 1 && n <= 1000000
}

type AccountResult struct {
	State     string    `json:"state"`
	Reason    string    `json:"reason"`
	Processed time.Time `json:"processedAt"`
}
