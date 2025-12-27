package domain

type PlayerRepository interface {
	Get(name string) (*Player, bool)
	Save(player *Player) error
	Delete(name string) error
	List() []*Player
	Count() int
	Reset()
}

type MatchRepository interface {
	Add(match *Match) error
	List() []*Match
	Count() int
	TrimOldest() bool
	Reset()
}
