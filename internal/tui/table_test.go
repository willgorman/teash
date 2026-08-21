package tui

import "testing"

func TestBuildColumns(t *testing.T) {
	tests := []struct {
		name        string
		labelKeys   []string
		width       int
		wantLabelW  int
		wantNumCols int
	}{
		{
			name:        "no label columns",
			labelKeys:   nil,
			width:       120,
			wantNumCols: 3,
		},
		{
			name:        "few labels fit comfortably",
			labelKeys:   []string{"env", "region"},
			width:       120,
			wantLabelW:  21, // (120 - 30 - 16 - 28 - 4) / 2
			wantNumCols: 5,
		},
		{
			name:        "many labels forced to the floor",
			labelKeys:   []string{"env", "region", "category1", "category2", "category3", "team"},
			width:       120,
			wantLabelW:  minLabelWidth, // computed perCol (7) is below the floor
			wantNumCols: 9,
		},
		{
			name:        "zero width falls back to floor",
			labelKeys:   []string{"env"},
			width:       0,
			wantLabelW:  minLabelWidth,
			wantNumCols: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := buildColumns(tt.labelKeys, tt.width)
			if len(cols) != tt.wantNumCols {
				t.Fatalf("got %d columns, want %d", len(cols), tt.wantNumCols)
			}
			for _, key := range tt.labelKeys {
				found := false
				for _, c := range cols {
					if c.Key() == key {
						found = true
						if tt.wantLabelW != 0 && c.Width() != tt.wantLabelW {
							t.Errorf("column %q width = %d, want %d", key, c.Width(), tt.wantLabelW)
						}
					}
				}
				if !found {
					t.Errorf("expected column %q not found", key)
				}
			}
		})
	}
}
