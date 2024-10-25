package internal

type (
	// Board represents Grafana dashboard.
	Board struct {
		ID     uint     `mapstructure:"id,omitempty"`
		UID    string   `mapstructure:"uid,omitempty"`
		Title  string   `mapstructure:"title"`
		Tags   []string `mapstructure:"tags"`
		Panels []*Panel `mapstructure:"panels"`
	}
	Panel struct {
		ID      uint      `mapstructure:"id"`
		OfType  panelType `mapstructure:"-"`     // it required for defining type of the panel
		Title   string    `mapstructure:"title"` // general
		Type    string    `mapstructure:"type"`
		Targets []Target  `mapstructure:"targets,omitempty"`
	}
	Target struct {
		Expr string `mapstructure:"expr,omitempty"`
	}
	panelType int8
)

type DatasourceRef struct {
	Type       string `mapstructure:"type"`
	UID        string `mapstructure:"UID"`
	LegacyName string `mapstructure:"-"`
}
