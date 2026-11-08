// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Script} from "forge-std/Script.sol";
import "forge-std/console.sol";
import {RiskReportOracle} from "../src/RiskReportOracle.sol";

/// @title DeployRiskReportOracle
/// @notice Deploy the RiskReportOracle contract and configure multi-oracle setup
contract DeployRiskReportOracle is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        uint256 oraclePrivateKey = vm.envOr("ORACLE_PRIVATE_KEY", uint256(0));

        vm.startBroadcast(deployerPrivateKey);

        RiskReportOracle oracle = new RiskReportOracle();
        console.log("RiskReportOracle deployed at:", address(oracle));

        if (oraclePrivateKey != 0) {
            address oracleAddr = vm.addr(oraclePrivateKey);
            oracle.addOracle(oracleAddr);
            console.log("Added oracle:", oracleAddr);
        }

        vm.stopBroadcast();

        console.log("---");
        console.log("Deployer (admin):", vm.addr(deployerPrivateKey));
        console.log("Contract address:", address(oracle));
        console.log("---");
        console.log("Verify with:");
        console.log(
            "forge verify-contract",
            address(oracle),
            "src/RiskReportOracle.sol:RiskReportOracle",
            "--chain",
            vm.envOr("CHAIN_ID", string("97")),
            "--verifier-url",
            "https://api-testnet.bscscan.com/api",
            "--etherscan-api-key",
            vm.envString("ETHERSCAN_API_KEY")
        );
    }
}
