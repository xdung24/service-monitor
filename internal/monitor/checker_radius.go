package monitor

import (
	"context"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"github.com/xdung24/conductor/internal/models"
)

// RadiusChecker sends a RADIUS Access-Request to the target server and
// considers the monitor UP when the server responds (either Access-Accept or
// Access-Reject).  A timeout or network error is reported as DOWN.
//
// Monitor fields used:
//
//	URL            – host:port of the RADIUS server (default port 1812)
//	HTTPUsername   – User-Name attribute value
//	HTTPPassword   – User-Password attribute value (cleartext, encrypted by protocol)
//	RadiusSecret   – RADIUS shared secret (required)
//	RadiusCalledStationID – optional Called-Station-Id attribute value
type RadiusChecker struct{}

// Check sends a RADIUS Access-Request and reports the result.
func (c *RadiusChecker) Check(ctx context.Context, m *models.Monitor) Result {
	if m.RadiusSecret == "" {
		return Result{Status: 0, Message: "radius_secret is required"}
	}

	addr, _, err := hostPort(m.URL, "1812")
	if err != nil {
		return Result{Status: 0, Message: "invalid host: " + err.Error()}
	}
	// Reconstruct full address including port so Exchange can dial it.
	_, port, err2 := hostPort(m.URL, "1812")
	if err2 != nil {
		return Result{Status: 0, Message: "invalid port: " + err2.Error()}
	}
	target := addr + ":" + port

	packet := radius.New(radius.CodeAccessRequest, []byte(m.RadiusSecret))
	if err := rfc2865.UserName_SetString(packet, m.HTTPUsername); err != nil {
		return Result{Status: 0, Message: "set User-Name: " + err.Error()}
	}
	if err := rfc2865.UserPassword_SetString(packet, m.HTTPPassword); err != nil {
		return Result{Status: 0, Message: "set User-Password: " + err.Error()}
	}
	if m.RadiusCalledStationID != "" {
		if err := rfc2865.CalledStationID_SetString(packet, m.RadiusCalledStationID); err != nil {
			return Result{Status: 0, Message: "set Called-Station-Id: " + err.Error()}
		}
	}

	response, err := radius.Exchange(ctx, packet, target)
	if err != nil {
		return Result{Status: 0, Message: "RADIUS exchange: " + err.Error()}
	}

	// Access-Accept = 2, Access-Reject = 3, Access-Challenge = 11
	// Any response means the server is reachable and functioning.
	switch response.Code {
	case radius.CodeAccessAccept:
		return Result{Status: 1, Message: "Access-Accept"}
	case radius.CodeAccessReject:
		return Result{Status: 1, Message: "Access-Reject (server reachable)"}
	case radius.CodeAccessChallenge:
		return Result{Status: 1, Message: "Access-Challenge (server reachable)"}
	default:
		return Result{Status: 1, Message: "RADIUS response: " + response.Code.String()}
	}
}
