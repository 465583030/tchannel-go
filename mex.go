package tchannel

import (
	"errors"
	"sync"

	"golang.org/x/net/context"
)

var (
	errDuplicateMex   = errors.New("multiple attempts to use the message id")
	errMexChannelFull = NewSystemError(ErrCodeBusy, "cannot send frame to message exchange channel")
)

const (
	messageExchangeSetInbound  = "inbound"
	messageExchangeSetOutbound = "outbound"
)

// A messageExchange tracks this channel's side of a message exchange with a peer.  Each
// message exchange has a channel that can be used to receive frames from the peer, and
// a Context that can controls when the exchange has timed out or been cancelled.
type messageExchange struct {
	recvCh  chan *Frame
	ctx     context.Context
	msgID   uint32
	msgType messageType
}

// forwardPeerFrame forwards a frame from a peer to the message exchange, where it can
// be pulled by whatever application thread is handling the exchange
func (mex *messageExchange) forwardPeerFrame(frame *Frame) error {
	select {
	case mex.recvCh <- frame:
		return nil
	default:
		return errMexChannelFull
	}
}

// messageExchangeSet manages a set of active message exchanges.  It is mainly
// used to route frames from a peer to the appropriate messageExchange, or to cancel
// or mark a messageExchange as being in error.  Each Connection maintains two
// messageExchangeSets, one to manage exchanges that it has initiated (outgoing), and another
// to manage exchanges that the peer has initiated (incoming).  The message-type specific handlers
// are responsible for ensuring that their message exchanges are properly registered and removed
// from the corresponding exchange set, but a background garbage collector also runs to find
// exchanges that have timed out or been cancelled without having been removed, to ensure that
// even buggy code doesn't result in out of memory situations
type messageExchangeSet struct {
	log       Logger
	name      string
	exchanges map[uint32]*messageExchange
	mut       sync.Mutex
}

// newExchange creates and adds a new message exchange to this set
func (mexset *messageExchangeSet) newExchange(ctx context.Context,
	msgType messageType, msgID uint32, bufferSize int) (*messageExchange, error) {
	mexset.log.Debugf("Creating new %s message exchange for [%s:%d]", mexset.name, msgType, msgID)

	mex := &messageExchange{
		msgType: msgType,
		msgID:   msgID,
		ctx:     ctx,
		recvCh:  make(chan *Frame, bufferSize),
	}

	mexset.mut.Lock()
	defer mexset.mut.Unlock()

	if existingMex := mexset.exchanges[mex.msgID]; existingMex != nil {
		if existingMex == mex {
			mexset.log.Warnf("%s mex for %s, %d registered multiple times",
				mexset.name, mex.msgType, mex.msgID)
		} else {
			mexset.log.Warnf("msg id %d used for both active mex %s and new mex %s",
				mex.msgID, existingMex.msgType, mex.msgType)
		}

		return nil, errDuplicateMex
	}

	mexset.exchanges[mex.msgID] = mex

	// TODO(mmihic): Put into a deadline ordered heap so we can garbage collected expired exchanges
	return mex, nil
}

// removeExchange removes a messge exchange from the set, if it exists.  It's perfectly
// fine to try and remove an exchange that has already completed
func (mexset *messageExchangeSet) removeExchange(msgID uint32) {
	mexset.log.Debugf("Removing %s message exchange %d", mexset.name, msgID)

	mexset.mut.Lock()
	defer mexset.mut.Unlock()

	delete(mexset.exchanges, msgID)
}

// forwardPeerFrame forwards a frame from the peer to the appropriate message exchange
// TODO(mmihic): We need to take the messageID here due to the weird originalMessageID field in error
// frame.  If we instead made the error frame message ID match the ID of the message in error, we could
// drop this additional parameters
func (mexset *messageExchangeSet) forwardPeerFrame(messageID uint32, frame *Frame) error {
	mexset.log.Debugf("forwarding %s %s", mexset.name, frame.Header)

	mexset.mut.Lock()
	mex := mexset.exchanges[messageID]
	mexset.mut.Unlock()

	if mex == nil {
		// This is ok since the exchange might have expired or been cancelled
		mexset.log.Warnf("received frame %s for message exchange that no longer exists", frame.Header)
		return nil
	}

	return mex.forwardPeerFrame(frame)
}
