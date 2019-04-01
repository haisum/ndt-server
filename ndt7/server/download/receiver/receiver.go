// Package receiver implements the counter-flow messages receiver.
package receiver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/m-lab/ndt-server/logging"
	"github.com/m-lab/ndt-server/ndt7/model"
	"github.com/m-lab/ndt-server/ndt7/spec"
)

func loop(ctx context.Context, conn *websocket.Conn, dst chan<- model.Measurement) {
	logging.Logger.Debug("receiver: start")
	defer logging.Logger.Debug("receiver: stop")
	defer close(dst)
	conn.SetReadLimit(spec.MinMaxMessageSize)
	receiverctx, cancel := context.WithTimeout(ctx, spec.MaxRuntime)
	defer cancel()
	for {
		select {
		case <-receiverctx.Done(): // Liveness!
			logging.Logger.Debug("receiver: context done")
			return
		default:
			// FALLTHROUGH
		}
		conn.SetReadDeadline(time.Now().Add(spec.MaxRuntime)) // Liveness!
		mtype, mdata, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			logging.Logger.WithError(err).Warn("receiver: conn.ReadMessage failed")
			return
		}
		if mtype != websocket.TextMessage {
			logging.Logger.Warn("receiver: got non-Text message")
			return
		}
		var measurement model.Measurement
		err = json.Unmarshal(mdata, &measurement)
		if err != nil {
			logging.Logger.WithError(err).Warn("receiver: json.Unmarshal failed")
			return
		}
		dst <- measurement // Liveness: this is blocking
	}
}

// Start starts the receiver in a background goroutine and returns the
// messages received from the client in the returned channel.
//
// Liveness guarantee: the goroutine will always terminate after a
// MaxRuntime timeout, provided that the consumer will keep reading
// from the returned channel.
func Start(ctx context.Context, conn *websocket.Conn) <-chan model.Measurement {
	dst := make(chan model.Measurement)
	go loop(ctx, conn, dst)
	return dst
}
