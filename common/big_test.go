package common

import (
	"database/sql/driver"
	"math/big"
	"reflect"
	"testing"
)

func TestBig_Scan(t *testing.T) {
	type args struct {
		src interface{}
	}
	tests := []struct {
		name    string
		args    args
		value   Big
		wantErr bool
	}{
		{
			name:    "scan int64",
			args:    args{src: int64(-10)},
			value:   Big(*big.NewInt(-10)),
			wantErr: false,
		},
		{
			name:    "scan uint64",
			args:    args{src: uint64(10)},
			value:   Big(*big.NewInt(10)),
			wantErr: false,
		},
		{
			name:    "scan bytes",
			args:    args{src: []byte{0x0a}},
			value:   Big(*big.NewInt(10)),
			wantErr: false,
		},
		{
			name:    "scan string",
			args:    args{src: "-10"},
			value:   Big(*big.NewInt(-10)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Big{}
			if err := b.Scan(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Big.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(*b, tt.value) {
					t.Errorf(
						"Big.Scan() wrong value (got: %v, want: %v)",
						*b, tt.value,
					)
				}
			}
		})
	}

}
func TestBig_Value(t *testing.T) {
	r := "12345"
	b := Big(*big.NewInt(12345))
	tests := []struct {
		name    string
		b       Big
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "working value",
			b:       b,
			want:    r,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Big.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Hash.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}
