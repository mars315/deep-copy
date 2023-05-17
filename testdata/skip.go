package testdata

type (
	// Basic .
	Basic struct {
		Level int32
	}

	// Player .
	Player struct {
		PlayerID  int64  `bson:"_id"`   //
		SessionID int64  `bson:"-"`     //
		Ctl       Ctl    `deepcopy:"-"` //
		Basic     *Basic //
	}

	// Ctl .
	Ctl interface {
		GetOpenTime() int64
	}
)
