package htlcswitch

import (
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/roasbeef/btcd/chaincfg/chainhash"
	"github.com/roasbeef/btcd/wire"
)

// InvoiceDatabase is an interface which represents the persistent subsystem
// which may search, lookup and settle invoices.
type InvoiceDatabase interface {
	// LookupInvoice attempts to look up an invoice according to its 32
	// byte payment hash.
	LookupInvoice(chainhash.Hash) (channeldb.Invoice, error)

	// SettleInvoice attempts to mark an invoice corresponding to the
	// passed payment hash as fully settled.
	SettleInvoice(chainhash.Hash) error
}

// ChannelLink is an interface which represents the subsystem for managing the
// incoming htlc requests, applying the changes to the channel, and also
// propagating/forwarding it to htlc switch.
//
//  abstraction level
//       ^
//       |
//       | - - - - - - - - - - - - Lightning - - - - - - - - - - - - -
//       |
//       | (Switch)		     (Switch)		       (Switch)
//       |  Alice <-- channel link --> Bob <-- channel link --> Carol
//	 |
//       | - - - - - - - - - - - - - TCP - - - - - - - - - - - - - - -
//       |
//       |  (Peer) 		     (Peer)	                (Peer)
//       |  Alice <----- tcp conn --> Bob <---- tcp conn -----> Carol
//       |
//
type ChannelLink interface {
	// TODO(roasbeef): modify interface to embed mail boxes?

	// HandleSwitchPacket handles the switch packets. This packets might be
	// forwarded to us from another channel link in case the htlc update
	// came from another peer or if the update was created by user
	// initially.
	//
	// NOTE: This function MUST be non-blocking (or block as little as
	// possible).
	HandleSwitchPacket(*htlcPacket)

	// HandleChannelUpdate handles the htlc requests as settle/add/fail
	// which sent to us from remote peer we have a channel with.
	//
	// NOTE: This function MUST be non-blocking (or block as little as
	// possible).
	HandleChannelUpdate(lnwire.Message)

	// ChanID returns the channel ID for the channel link. The channel ID
	// is a more compact representation of a channel's full outpoint.
	ChanID() lnwire.ChannelID

	// ShortChanID returns the short channel ID for the channel link. The
	// short channel ID encodes the exact location in the main chain that
	// the original funding output can be found.
	ShortChanID() lnwire.ShortChannelID

	// UpdateShortChanID updates the short channel ID for a link. This may
	// be required in the event that a link is created before the short
	// chan ID for it is known, or a re-org occurs, and the funding
	// transaction changes location within the chain.
	UpdateShortChanID(lnwire.ShortChannelID)

	// UpdateForwardingPolicy updates the forwarding policy for the target
	// ChannelLink. Once updated, the link will use the new forwarding
	// policy to govern if it an incoming HTLC should be forwarded or not.
	UpdateForwardingPolicy(ForwardingPolicy)

	// Bandwidth returns the amount of milli-satoshis which current link
	// might pass through channel link. The value returned from this method
	// represents the up to date available flow through the channel. This
	// takes into account any forwarded but un-cleared HTLC's, and any
	// HTLC's which have been set to the over flow queue.
	Bandwidth() lnwire.MilliSatoshi

	// Stats return the statistics of channel link. Number of updates,
	// total sent/received milli-satoshis.
	Stats() (uint64, lnwire.MilliSatoshi, lnwire.MilliSatoshi)

	// Peer returns the representation of remote peer with which we have
	// the channel link opened.
	Peer() Peer

	// EligibleToForward returns a bool indicating if the channel is able
	// to actively accept requests to forward HTLC's. A channel may be
	// active, but not able to forward HTLC's if it hasn't yet finalized
	// the pre-channel operation protocol with the remote peer. The switch
	// will use this function in forwarding decisions accordingly.
	EligibleToForward() bool

	// Start/Stop are used to initiate the start/stop of the channel link
	// functioning.
	Start() error
	Stop()
}

// Peer is an interface which represents the remote lightning node inside our
// system.
type Peer interface {
	// SendMessage sends message to remote peer.
	SendMessage(lnwire.Message) error

	// WipeChannel removes the channel uniquely identified by its channel
	// point from all indexes associated with the peer.
	WipeChannel(*wire.OutPoint) error

	// PubKey returns the serialize public key of the source peer.
	PubKey() [33]byte

	// Disconnect disconnects with peer if we have error which we can't
	// properly handle.
	Disconnect(reason error)
}
