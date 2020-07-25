package builtInFunctions

import (
	"encoding/hex"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var _ process.BuiltinFunction = (*saveUserName)(nil)

const userNameHashLength = 32

type saveUserName struct {
	gasCost         uint64
	mapDnsAddresses map[string]struct{}
	enableChange    bool
}

// NewSaveUserNameFunc returns a username built in function implementation
func NewSaveUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	enableChange bool,
) (*saveUserName, error) {
	if mapDnsAddresses == nil {
		return nil, process.ErrNilDnsAddresses
	}

	s := &saveUserName{
		gasCost:      gasCost,
		enableChange: enableChange,
	}
	s.mapDnsAddresses = make(map[string]struct{}, len(mapDnsAddresses))
	for key := range mapDnsAddresses {
		s.mapDnsAddresses[key] = struct{}{}
	}

	return s, nil
}

// ProcessBuiltinFunction sets the username to the account if it is allowed
func (s *saveUserName) ProcessBuiltinFunction(
	_, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, process.ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, process.ErrBuiltInFunctionCalledWithValue
	}
	if vmInput.GasProvided < s.gasCost {
		return nil, process.ErrNotEnoughGas
	}
	_, ok := s.mapDnsAddresses[string(vmInput.CallerAddr)]
	if !ok {
		return nil, process.ErrCallerIsNotTheDNSAddress
	}
	if len(vmInput.Arguments) != 1 || len(vmInput.Arguments[0]) != userNameHashLength {
		return nil, process.ErrInvalidArguments
	}

	if check.IfNil(acntDst) {
		log.Warn("setUserName called dst not in shard")
		// cross-shard call, in sender shard only the gas is taken out
		vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		setUserNameTxData := core.BuiltInFunctionSetUserName + "@" + hex.EncodeToString(vmInput.Arguments[0])
		vmOutput.OutputAccounts[string(vmInput.RecipientAddr)] = &vmcommon.OutputAccount{
			Address:  vmInput.RecipientAddr,
			Data:     []byte(setUserNameTxData),
			CallType: vmcommon.AsynchronousCall,
			GasLimit: vmInput.GasProvided,
		}
		return vmOutput, nil
	}

	log.Warn("setUserName called")
	currentUserName := acntDst.GetUserName()
	if !s.enableChange && len(currentUserName) > 0 {
		return nil, process.ErrUserNameChangeIsDisabled
	}

	acntDst.SetUserName(vmInput.Arguments[0])

	return &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided - s.gasCost}, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (s *saveUserName) IsInterfaceNil() bool {
	return s == nil
}
