package api

import (
	"context"
	"net/http"
	"time"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
)

func (a *Api) QRCodeHandler(w http.ResponseWriter, r *http.Request) {
	if a.qrCancel != nil {
		a.qrCancel()
	}
	l := logger.With("api", "qrcode")
	id, image, err := common.QRCode()
	if err != nil {
		l.Error(err)
		responseServerError(w, err)
		return
	}

	w.Header().Add("Content-Type", common.MimeImage)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(image)
	if err != nil {
		l.Error("write content fail: ", err)
	}

	l.Debug("response image done")

	go func() {
		const timeout = time.Second * 60
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		a.qrCancel = cancel
		defer func() {
			a.qrCancel()
			a.qrCancel = nil
		}()

		alt, err := common.CheckScanState(ctx, id)
		if err != nil {
			l.Error(err)
			responseServerError(w, err)
			return
		}
		cookie, err := common.GetCookie(alt)
		if err != nil {
			l.Error(err)
			return
		}
		if a.cookieAcceptor != nil {
			a.cookieAcceptor.Accept(cookie)
		}
	}()
}
