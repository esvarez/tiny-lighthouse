package sim

// BoatKind identifies one of the boat types from the design document.
type BoatKind int

const (
	Fishing BoatKind = iota
	Merchant
	Speedboat
	Ferry
	Rowboat
)

// BoatSpec holds the tuning values for a boat type.
type BoatSpec struct {
	Name      string
	Speed     float64 // px/s
	Turn      float64 // rad/s
	Radius    float64 // hull radius in px
	Lookahead float64 // how far ahead the captain checks for known rocks
	Reward    int     // coins paid on docking
	RepCost   int     // hearts lost on a wreck
}

var Specs = [...]BoatSpec{
	Fishing:   {Name: "Fishing boat", Speed: 62, Turn: 2.3, Radius: 7, Lookahead: 115, Reward: 5, RepCost: 1},
	Merchant:  {Name: "Merchant ship", Speed: 80, Turn: 1.3, Radius: 12, Lookahead: 170, Reward: 12, RepCost: 1},
	Speedboat: {Name: "Speedboat", Speed: 140, Turn: 3.2, Radius: 6, Lookahead: 75, Reward: 10, RepCost: 1},
	Ferry:     {Name: "Ferry", Speed: 52, Turn: 1.5, Radius: 11, Lookahead: 130, Reward: 25, RepCost: 2},
	Rowboat:   {Name: "Lost rowboat", Speed: 48, Turn: 2.6, Radius: 5, Lookahead: 90, Reward: 15, RepCost: 1},
}

// NightConfig describes what a single night throws at the player.
type NightConfig struct {
	Boats    int
	Kinds    []BoatKind // kinds that may appear
	New      []BoatKind // kinds introduced tonight (guaranteed to appear twice)
	Rocks    int
	Logs     int
	Fogs     int
	Storm    bool
	OilLimit bool
	Intro    string
}

// Nights follows the progression table of the design document.
var Nights = [...]NightConfig{
	{Boats: 4, Kinds: []BoatKind{Fishing}, Rocks: 4, Intro: "Boats sail to the lighthouse and turn toward your beam. Guide them past the rocks."},
	{Boats: 6, Kinds: []BoatKind{Fishing}, Rocks: 5, OilLimit: true, Intro: "Oil now runs out. Space turns the lamp off to save it."},
	{Boats: 7, Kinds: []BoatKind{Fishing, Merchant}, New: []BoatKind{Merchant}, Rocks: 6, OilLimit: true, Intro: "Merchant ships are wide and turn slowly. Warn them early."},
	{Boats: 8, Kinds: []BoatKind{Fishing, Merchant}, Rocks: 6, Fogs: 2, OilLimit: true, Intro: "Light fog: the beam reaches less far inside it."},
	{Boats: 9, Kinds: []BoatKind{Fishing, Merchant, Speedboat}, New: []BoatKind{Speedboat}, Rocks: 7, Fogs: 2, OilLimit: true, Intro: "Speedboats react late. Aim quickly."},
	{Boats: 10, Kinds: []BoatKind{Fishing, Merchant, Speedboat}, Rocks: 6, Logs: 3, Fogs: 2, OilLimit: true, Intro: "Drifting logs move across the bay."},
	{Boats: 10, Kinds: []BoatKind{Fishing, Merchant, Speedboat, Ferry}, New: []BoatKind{Ferry}, Rocks: 7, Logs: 2, Fogs: 2, OilLimit: true, Intro: "The ferry carries the whole town. Losing it costs 2 hearts."},
	{Boats: 11, Kinds: []BoatKind{Fishing, Merchant, Speedboat, Ferry}, Rocks: 7, Logs: 2, Fogs: 1, Storm: true, OilLimit: true, Intro: "First storm: wind pushes boats and lightning damages the tower."},
	{Boats: 12, Kinds: []BoatKind{Fishing, Merchant, Speedboat, Ferry, Rowboat}, New: []BoatKind{Rowboat}, Rocks: 7, Logs: 2, Fogs: 5, OilLimit: true, Intro: "Heavy fog. Lost rowboats only see rocks while you keep them in the light."},
	{Boats: 14, Kinds: []BoatKind{Fishing, Merchant, Speedboat, Ferry, Rowboat}, Rocks: 8, Logs: 4, Fogs: 3, Storm: true, OilLimit: true, Intro: "The final storm. Everything at once."},
}

// UpgradeID indexes Upgrades.
type UpgradeID int

const (
	BiggerTank UpgradeID = iota
	PolishedLens
	WideReflector
	FasterGears
	Foghorn
	SecondLamp
	StormShutters
	HarborBuoy
	numUpgrades
)

type UpgradeInfo struct {
	Name   string
	Cost   int
	Effect string
	Max    int // how many times it can be bought
}

var Upgrades = [numUpgrades]UpgradeInfo{
	BiggerTank:    {"Bigger oil tank", 20, "+50% oil capacity", 1},
	PolishedLens:  {"Polished lens", 30, "+25% beam range", 1},
	WideReflector: {"Wide reflector", 35, "Wider beam angle", 1},
	FasterGears:   {"Faster gears", 25, "Beam rotates 30% faster", 1},
	Foghorn:       {"Foghorn", 40, "Right click slows nearby boats (20 s cooldown)", 1},
	SecondLamp:    {"Second lamp", 60, "Fixed light; click the sea by day to place it", 1},
	StormShutters: {"Storm shutters", 30, "Halves storm damage", 1},
	HarborBuoy:    {"Harbor buoy", 15, "Permanently marks one rock (stacks)", 3},
}

// story is the captain's morning note after each night; it slowly hints at
// why the old keeper left.
var story = [...]string{
	"Old Maren kept this light for forty years. Nobody knows why she left so suddenly.",
	"Maren's logbook still sits by the lamp. The last pages are torn out.",
	"The merchants say Maren refused a bribe to let a certain ship run aground.",
	"Someone saw a light on the north cliffs last night. Nobody lives up there.",
	"The harbor master asked about you today. He seemed... disappointed you are doing well.",
	"A torn page washed ashore: 'they want the bay dark, so the salvage pays.'",
	"The ferry captain swears Maren is alive and hiding in the old fishing hut.",
	"After the storm, a note was nailed to your door: 'Keep it burning. — M.'",
	"The harbor master's salvage boat has been seen waiting near the rocks every night.",
	"",
}

var thanks = [...]string{
	"Every boat came home. The town sends warm bread.",
	"Not a single splinter lost. The fishers raise a glass to you.",
}

var complaints = [...]string{
	"We lost boats out there. The tavern was quiet tonight.",
	"Some captains are grumbling. Light the rocks sooner.",
}
