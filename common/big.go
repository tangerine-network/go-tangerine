// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)
)

// Big support database/sql Scan and Value.
type Big big.Int

// Scan implements Scanner for database/sql.
func (b *Big) Scan(src interface{}) error {
	newB := new(big.Int)
	switch t := src.(type) {
	case int64:
		*b = Big(*newB.SetInt64(t))
	case uint64:
		*b = Big(*newB.SetUint64(t))
	case []byte:
		*b = Big(*newB.SetBytes(t))
	case string:
		v, ok := newB.SetString(t, 10)
		if !ok {
			return fmt.Errorf("invalid string format %v", src)
		}
		*b = Big(*v)
	default:
		return fmt.Errorf("can't scan %T into Big", src)
	}
	return nil
}

// Value implements valuer for database/sql.
func (b Big) Value() (driver.Value, error) {
	b2 := big.Int(b)
	return (&b2).String(), nil
}

// String returns decimal value in string type.
func (b *Big) String() string { return (*big.Int)(b).String() }

// BigInt convert common.Big to big.Int.
func (b *Big) BigInt() *big.Int { return (*big.Int)(b) }
