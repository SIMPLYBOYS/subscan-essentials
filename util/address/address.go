package address

import (
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
)

func SS58Address(address string) string {
	return ss58.Encode(address, util.StringToInt(util.AddressType))
}
