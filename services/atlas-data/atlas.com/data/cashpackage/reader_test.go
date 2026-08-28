package cashpackage

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

const testXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="CashPackage.img">
  <imgdir name="9100000">
    <imgdir name="SN">
      <int name="0" value="10000"/>
      <int name="1" value="10001"/>
      <int name="2" value="10002"/>
    </imgdir>
  </imgdir>
  <imgdir name="9100001">
    <imgdir name="SN">
      <int name="0" value="20000"/>
    </imgdir>
  </imgdir>
  <imgdir name="9100002">
  </imgdir>
</imgdir>
`

func TestReadCashPackages(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms, err := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))()
	if err != nil {
		t.Fatal(err)
	}

	if len(rms) != 3 {
		t.Fatalf("len(rms) = %d, want 3", len(rms))
	}

	tests := []struct {
		name string
		want RestModel
	}{
		{name: "package with three serial numbers", want: RestModel{Id: 9100000, SerialNumbers: []uint32{10000, 10001, 10002}}},
		{name: "package with one serial number", want: RestModel{Id: 9100001, SerialNumbers: []uint32{20000}}},
		{name: "package with no serial numbers", want: RestModel{Id: 9100002, SerialNumbers: []uint32{}}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rms[i]
			if got.Id != tt.want.Id {
				t.Fatalf("rms[%d].Id = %d, want %d", i, got.Id, tt.want.Id)
			}
			if got.SerialNumbers == nil {
				t.Fatalf("rms[%d].SerialNumbers = nil, want non-nil (possibly empty) slice", i)
			}
			if len(got.SerialNumbers) != len(tt.want.SerialNumbers) {
				t.Fatalf("rms[%d].SerialNumbers = %v, want %v", i, got.SerialNumbers, tt.want.SerialNumbers)
			}
			for j, sn := range tt.want.SerialNumbers {
				if got.SerialNumbers[j] != sn {
					t.Fatalf("rms[%d].SerialNumbers[%d] = %d, want %d", i, j, got.SerialNumbers[j], sn)
				}
			}
		})
	}
}
