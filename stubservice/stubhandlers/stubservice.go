package stubhandlers

import (
	"fmt"
	"log"
	"net/http"

	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/pkg/errors"
)

// StubService serves redirects or modified stubs
type StubService struct {
	Handler StubHandler

	AttributionCodeValidator *attributioncode.Validator

	RavenClient *raven.Client
}

func (s *StubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))
		http.Redirect(w, req, backupURL, http.StatusFound)
	}

	handleError := func(err error) {
		log.Println(err)
		if s.RavenClient != nil {
			raven.CaptureMessage(fmt.Sprintf("%v", err), map[string]string{
				"url": req.URL.String(),
			})
		}
		redirectBouncer()
	}

	code, err := s.AttributionCodeValidator.Validate(query.Get("attribution_code"), query.Get("attribution_sig"))
	if err != nil {
		handleError(errors.Wrap(err, "could not validate attribution_code"))
		return
	}

	err = s.Handler.ServeStub(w, req, code.Encode())
	if err != nil {
		handleError(errors.Wrap(err, "ServeStub"))
		return
	}
}
