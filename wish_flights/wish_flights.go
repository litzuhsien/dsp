package wish_flights

import (
	"encoding/json"
	"fmt"
	"github.com/clixxa/dsp/bindings"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type WinFlight struct {
	Runtime struct {
		Storage struct {
			Purchases func([18]interface{}, *error)
			Recall    func(json.Unmarshaler, *error, string)
		}
		Logger *log.Logger
		Debug  *log.Logger
	} `json:"-"`

	HttpRequest  *http.Request       `json:"-"`
	HttpResponse http.ResponseWriter `json:"-"`

	RevTXHome int    `json:"-"`
	PaidPrice int    `json:"-"`
	RecallID  string `json:"-"`
	SaleID    int    `json:"-"`

	FolderID   int     `json:"folder"`
	CreativeID int     `json:"creative"`
	Margin     int     `json:"margin"`
	Request    Request `json:"req"`

	StartTime time.Time
	Error     error `json:"-"`
}

func (wf *WinFlight) String() string {
	e := ""
	if wf.Error != nil {
		e = wf.Error.Error()
	}
	return fmt.Sprintf(`winflight id%d err%s`, wf.RecallID, e)
}

func (wf *WinFlight) Launch() {
	defer func() {
		if err := recover(); err != nil {
			wf.Runtime.Logger.Println("uncaught panic, stack trace following", err)
			s := debug.Stack()
			wf.Runtime.Logger.Println(string(s))
		}
	}()
	ReadWinNotice(wf)
	ProcessWin(wf)
	WriteWinResponse(wf)
}

func (wf *WinFlight) Columns() [18]interface{} {
	return [18]interface{}{wf.SaleID, !wf.Request.Test, wf.RevTXHome, wf.RevTXHome, wf.PaidPrice, wf.PaidPrice, 0, wf.FolderID, wf.CreativeID, wf.Request.Device.Geo.CountryID, wf.Request.Site.VerticalID, wf.Request.Site.BrandID, wf.Request.Site.NetworkID, wf.Request.Site.SubNetworkID, wf.Request.Site.NetworkTypeID, wf.Request.User.GenderID, wf.Request.Device.DeviceTypeID}
}

type wfProxy WinFlight

func (wf *WinFlight) UnmarshalJSON(d []byte) error {
	return json.Unmarshal(d, (*wfProxy)(wf))
}

func ReadWinNotice(flight *WinFlight) {
	flight.StartTime = time.Now()
	flight.Runtime.Logger.Println(`starting ProcessWin`, flight.String())

	if u, e := url.ParseRequestURI(flight.HttpRequest.RequestURI); e != nil {
		flight.Runtime.Logger.Println(`win url not valid`, e.Error())
	} else {
		flight.RecallID = u.Query().Get("key")
		flight.Runtime.Logger.Printf(`got recallid %s`, flight.RecallID)

		p := u.Query().Get("price")
		if price, e := strconv.ParseInt(p, 10, 64); e != nil {
			flight.Runtime.Logger.Println(`win url not valid`, e.Error())
		} else {
			flight.PaidPrice = int(price)
			flight.Runtime.Logger.Printf(`got price %d`, flight.PaidPrice)
		}

		imp := u.Query().Get("imp")
		if impid, e := strconv.ParseInt(imp, 10, 64); e != nil {
			flight.Runtime.Logger.Println(`win url not valid`, e.Error())
		} else {
			flight.SaleID = int(impid)
			flight.Runtime.Logger.Printf(`got impid %d`, flight.SaleID)
		}
	}
}

// Perform any post-flight logging, etc
func ProcessWin(flight *WinFlight) {
	if flight.Error != nil {
		flight.Runtime.Logger.Println(`not processing win because err: %s`, flight.Error.Error())
		return
	}

	flight.Runtime.Logger.Printf(`getting bid info for %d`, flight.RecallID)
	flight.Runtime.Storage.Recall(flight, &flight.Error, flight.RecallID)
	flight.RevTXHome = flight.PaidPrice + flight.Margin

	flight.Runtime.Logger.Printf(`adding margin of %d to paid price of %d`, flight.Margin, flight.PaidPrice)
	flight.Runtime.Logger.Printf(`win: revssp%d revtx%d`, flight.PaidPrice, flight.RevTXHome)
	flight.Runtime.Logger.Println(`inserting purchase record`)
	flight.Runtime.Storage.Purchases(flight.Columns(), &flight.Error)
}

func WriteWinResponse(flight *WinFlight) {
	if flight.Error != nil {
		flight.Runtime.Logger.Printf(`!! got an error handling win notice !! %s !!`, flight.Error.Error())
		flight.Runtime.Debug.Printf(`!! got an error handling win notice !! %s !!`, flight.Error.Error())
		flight.Runtime.Logger.Printf(`winflight %#v`, flight)
		flight.HttpResponse.WriteHeader(http.StatusInternalServerError)
	} else {
		flight.HttpResponse.WriteHeader(http.StatusOK)
	}
	flight.Runtime.Logger.Println(`dsp /win took`, time.Since(flight.StartTime))
}