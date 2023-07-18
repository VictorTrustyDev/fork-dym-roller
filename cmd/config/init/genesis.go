package initconfig

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/dymensionxyz/roller/cmd/consts"
	"github.com/dymensionxyz/roller/cmd/utils"
	"github.com/dymensionxyz/roller/config"

	"os/exec"
	"path/filepath"

	"github.com/tidwall/sjson"
)

func initializeRollappGenesis(initConfig config.RollappConfig) error {
	totalTokenSupply, success := new(big.Int).SetString(initConfig.TokenSupply, 10)
	if !success {
		return fmt.Errorf("invalid token supply")
	}
	totalTokenSupply = totalTokenSupply.Mul(totalTokenSupply, new(big.Int).Exp(big.NewInt(10),
		new(big.Int).SetUint64(uint64(initConfig.Decimals)), nil))
	relayerGenesisBalance := new(big.Int).Div(totalTokenSupply, big.NewInt(10))
	sequencerGenesisBalance := new(big.Int).Sub(totalTokenSupply, relayerGenesisBalance)
	sequencerBalanceStr := sequencerGenesisBalance.String() + initConfig.Denom
	relayerBalanceStr := relayerGenesisBalance.String() + initConfig.Denom
	rollappConfigDirPath := filepath.Join(initConfig.Home, consts.ConfigDirName.Rollapp)
	genesisSequencerAccountCmd := exec.Command(initConfig.RollappBinary, "add-genesis-account",
		consts.KeysIds.RollappSequencer, sequencerBalanceStr, "--keyring-backend", "test", "--home", rollappConfigDirPath)
	_, err := utils.ExecBashCommand(genesisSequencerAccountCmd)
	if err != nil {
		return err
	}
	rlyRollappAddress, err := utils.GetRelayerAddress(initConfig.Home, initConfig.RollappID)
	if err != nil {
		return err
	}
	genesisRelayerAccountCmd := exec.Command(initConfig.RollappBinary, "add-genesis-account",
		rlyRollappAddress, relayerBalanceStr, "--home", rollappConfigDirPath)
	_, err = utils.ExecBashCommand(genesisRelayerAccountCmd)
	if err != nil {
		return err
	}
	err = updateGenesisParams(GetGenesisFilePath(initConfig.Home), initConfig.Denom, initConfig.Decimals)
	if err != nil {
		return err
	}
	return nil
}

func GetGenesisFilePath(root string) string {
	return filepath.Join(RollappConfigDir(root), "genesis.json")
}

type PathValue struct {
	Path  string
	Value interface{}
}

type BankDenomMetadata struct {
	Base        string                  `json:"base"`
	DenomUnits  []BankDenomUnitMetadata `json:"denom_units"`
	Description string                  `json:"description"`
	Display     string                  `json:"display"`
	Name        string                  `json:"name"`
	Symbol      string                  `json:"symbol"`
}

type BankDenomUnitMetadata struct {
	Aliases  []string `json:"aliases"`
	Denom    string   `json:"denom"`
	Exponent uint     `json:"exponent"`
}

// TODO(#130): fix to support epochs
func getDefaultGenesisParams(denom string, decimals uint) []PathValue {
	displayDenom := denom[1:]

	return []PathValue{
		{"app_state.mint.params.mint_denom", denom},
		{"app_state.staking.params.bond_denom", denom},
		{"app_state.crisis.constant_fee.denom", denom},
		{"app_state.evm.params.evm_denom", denom},
		{"app_state.gov.deposit_params.min_deposit.0.denom", denom},
		{"consensus_params.block.max_gas", "40000000"},
		{"app_state.feemarket.params.no_base_fee", true},
		{"app_state.distribution.params.base_proposer_reward", "0.8"},
		{"app_state.distribution.params.community_tax", "0.00002"},
		{"app_state.gov.voting_params.voting_period", "300s"},
		{"app_state.staking.params.unbonding_time", "3628800s"},
		{"app_state.bank.denom_metadata", []BankDenomMetadata{
			{
				Base: denom,
				DenomUnits: []BankDenomUnitMetadata{
					{
						Aliases:  []string{},
						Denom:    denom,
						Exponent: 0,
					},
					{
						Aliases:  []string{},
						Denom:    displayDenom,
						Exponent: decimals,
					},
				},
				Description: fmt.Sprintf("Denom metadata for %s (%s)", displayDenom, denom),
				Display:     displayDenom,
				Name:        fmt.Sprintf("%s%s", strings.ToUpper(displayDenom[:1]), strings.ToLower(displayDenom[1:])),
				Symbol:      strings.ToUpper(displayDenom),
			},
		}},
	}
}

func UpdateJSONParams(jsonFilePath string, params []PathValue) error {
	jsonFileContent, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		return err
	}
	jsonFileContentString := string(jsonFileContent)
	for _, param := range params {
		jsonFileContentString, err = sjson.Set(jsonFileContentString, param.Path, param.Value)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(jsonFilePath, []byte(jsonFileContentString), 0644)
	if err != nil {
		return err
	}
	return nil
}

func updateGenesisParams(genesisFilePath string, denom string, decimals uint) error {
	params := getDefaultGenesisParams(denom, decimals)
	return UpdateJSONParams(genesisFilePath, params)
}
