// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {RiskReportOracle} from "../src/RiskReportOracle.sol";

contract RiskReportOracleTest is Test {
    RiskReportOracle public oracle;

    address public admin = address(0xAAA);
    address public oracle1 = address(0xBBB);
    address public oracle2 = address(0xCCC);
    address public user = address(0xDDD);
    address public token = address(0xDEAD);

    bytes32 public constant ORACLE_ROLE = keccak256("ORACLE_ROLE");

    function setUp() public {
        vm.prank(admin);
        oracle = new RiskReportOracle();

        vm.prank(admin);
        oracle.addOracle(oracle1);
    }

    // ==========================================
    // Constructor Tests
    // ==========================================

    function test_Constructor_AdminHasRoles() public view {
        assertTrue(oracle.hasRole(oracle.DEFAULT_ADMIN_ROLE(), admin));
        assertTrue(oracle.hasRole(ORACLE_ROLE, admin));
    }

    function test_Constructor_AdminIsOracle() public view {
        assertTrue(oracle.hasRole(ORACLE_ROLE, admin));
    }

    // ==========================================
    // submitReport Tests
    // ==========================================

    function test_SubmitReport_Success() public {
        bytes32 reportHash = keccak256("test report data");
        uint8 riskScore = 5;

        vm.prank(oracle1);
        oracle.submitReport(token, reportHash, riskScore, "v1.0.0");

        // Use getScanStats which returns a proper ReportRecord struct
        (, uint256 reports, RiskReportOracle.ReportRecord memory record) =
            oracle.getScanStats(token);

        assertEq(record.reportHash, reportHash);
        assertEq(record.riskScore, riskScore);
        assertEq(record.version, "v1.0.0");
        assertGt(record.timestamp, 0);
        assertEq(reports, 1);
    }

    function test_SubmitReport_IncrementsCount() public {
        bytes32 hash1 = keccak256("report 1");
        bytes32 hash2 = keccak256("report 2");

        vm.startPrank(oracle1);
        oracle.submitReport(token, hash1, 3, "v1.0.0");
        oracle.submitReport(token, hash2, 7, "v1.0.0");
        vm.stopPrank();

        assertEq(oracle.reportCount(token), 2);

        // Destructure the public getter tuple
        (bytes32 h, , , ) = oracle.latestReport(token);
        assertEq(h, hash2);
    }

    function test_SubmitReport_RevertIf_ZeroAddress() public {
        bytes32 reportHash = keccak256("test");
        vm.prank(oracle1);
        vm.expectRevert(RiskReportOracle.InvalidTokenAddress.selector);
        oracle.submitReport(address(0), reportHash, 5, "v1.0.0");
    }

    function test_SubmitReport_RevertIf_EmptyHash() public {
        vm.prank(oracle1);
        vm.expectRevert(RiskReportOracle.EmptyReportHash.selector);
        oracle.submitReport(token, bytes32(0), 5, "v1.0.0");
    }

    function test_SubmitReport_RevertIf_ScoreTooLow() public {
        bytes32 reportHash = keccak256("test");
        vm.prank(oracle1);
        vm.expectRevert(
            abi.encodeWithSelector(RiskReportOracle.InvalidRiskScore.selector, 0)
        );
        oracle.submitReport(token, reportHash, 0, "v1.0.0");
    }

    function test_SubmitReport_RevertIf_ScoreTooHigh() public {
        bytes32 reportHash = keccak256("test");
        vm.prank(oracle1);
        vm.expectRevert(
            abi.encodeWithSelector(RiskReportOracle.InvalidRiskScore.selector, 11)
        );
        oracle.submitReport(token, reportHash, 11, "v1.0.0");
    }

    function test_SubmitReport_RevertIf_EmptyVersion() public {
        bytes32 reportHash = keccak256("test");
        vm.prank(oracle1);
        vm.expectRevert(RiskReportOracle.EmptyVersion.selector);
        oracle.submitReport(token, reportHash, 5, "");
    }

    function test_SubmitReport_RevertIf_NotOracle() public {
        bytes32 reportHash = keccak256("unauthorized");
        vm.prank(user);
        vm.expectRevert(
            abi.encodeWithSelector(
                RiskReportOracle.Unauthorized.selector, user, ORACLE_ROLE
            )
        );
        oracle.submitReport(token, reportHash, 5, "v1.0.0");
    }

    // ==========================================
    // recordScan Tests
    // ==========================================

    function test_RecordScan_Success() public {
        vm.prank(user);
        oracle.recordScan(token);
        assertEq(oracle.scanCount(token), 1);

        vm.prank(user);
        oracle.recordScan(token);
        assertEq(oracle.scanCount(token), 2);
    }

    function test_RecordScan_RevertIf_ZeroAddress() public {
        vm.expectRevert(RiskReportOracle.InvalidTokenAddress.selector);
        oracle.recordScan(address(0));
    }

    // ==========================================
    // View Function Tests
    // ==========================================

    function test_GetLatestReport_EmptyForNewToken() public view {
        (bytes32 h, uint8 s, uint256 t, string memory v) = oracle.latestReport(token);
        assertEq(h, bytes32(0));
        assertEq(s, 0);
        assertEq(t, 0);
        assertEq(v, "");
    }

    function test_GetReportCount_ZeroForNewToken() public view {
        assertEq(oracle.getReportCount(token), 0);
    }

    function test_GetScanStats_ReturnsCorrectData() public {
        bytes32 reportHash = keccak256("stats test");
        vm.prank(oracle1);
        oracle.submitReport(token, reportHash, 8, "v1.0.0");

        vm.prank(user);
        oracle.recordScan(token);
        vm.prank(user);
        oracle.recordScan(token);

        (uint256 scans, uint256 reports, RiskReportOracle.ReportRecord memory lastReport) =
            oracle.getScanStats(token);

        assertEq(scans, 2);
        assertEq(reports, 1);
        assertEq(lastReport.reportHash, reportHash);
        assertEq(lastReport.riskScore, 8);
    }

    // ==========================================
    // Access Control Tests
    // ==========================================

    function test_GrantRole_Success() public {
        vm.prank(admin);
        oracle.grantRole(ORACLE_ROLE, oracle2);
        assertTrue(oracle.hasRole(ORACLE_ROLE, oracle2));
    }

    function test_AddOracle_Success() public {
        vm.prank(admin);
        oracle.addOracle(oracle2);
        assertTrue(oracle.hasRole(ORACLE_ROLE, oracle2));
    }

    function test_AddOracle_RevertIf_ZeroAddress() public {
        vm.prank(admin);
        vm.expectRevert(RiskReportOracle.InvalidOracleAddress.selector);
        oracle.addOracle(address(0));
    }

    function test_AddOracle_RevertIf_NotAdmin() public {
        vm.prank(user);
        vm.expectRevert();
        oracle.addOracle(oracle2);
    }

    function test_RemoveOracle_Success() public {
        vm.prank(admin);
        oracle.removeOracle(oracle1);
        assertFalse(oracle.hasRole(ORACLE_ROLE, oracle1));
    }

    function test_RenounceRole_Success() public {
        vm.prank(oracle1);
        oracle.renounceRole(ORACLE_ROLE, oracle1);
        assertFalse(oracle.hasRole(ORACLE_ROLE, oracle1));
    }

    // ==========================================
    // Event Tests
    // ==========================================

    function test_SubmitReport_EmitsEvent() public {
        bytes32 reportHash = keccak256("event test");

        vm.prank(oracle1);
        vm.expectEmit(true, true, false, true);
        emit RiskReportOracle.ReportSubmitted(token, reportHash, 5, block.timestamp, 1);
        oracle.submitReport(token, reportHash, 5, "v1.0.0");
    }

    function test_RecordScan_EmitsEvent() public {
        vm.prank(user);
        vm.expectEmit(true, true, false, true);
        emit RiskReportOracle.TokenScanned(token, user, 1);
        oracle.recordScan(token);
    }

    function test_RoleGranted_EmitsEvent() public {
        vm.prank(admin);
        vm.expectEmit(true, true, true, false);
        emit RiskReportOracle.RoleGranted(ORACLE_ROLE, oracle2, admin);
        oracle.grantRole(ORACLE_ROLE, oracle2);
    }

    // ==========================================
    // Score Boundary Tests
    // ==========================================

    function test_SubmitReport_ScoreMinBoundary() public {
        bytes32 hash = keccak256("boundary min");
        vm.prank(oracle1);
        oracle.submitReport(token, hash, 1, "v1.0.0");

        (, uint8 score, , ) = oracle.latestReport(token);
        assertEq(score, 1);
    }

    function test_SubmitReport_ScoreMaxBoundary() public {
        bytes32 hash = keccak256("boundary max");
        vm.prank(oracle1);
        oracle.submitReport(token, hash, 10, "v1.0.0");

        (, uint8 score, , ) = oracle.latestReport(token);
        assertEq(score, 10);
    }

    // ==========================================
    // Integration / Scenario Tests
    // ==========================================

    function test_FullWorkflow_MultipleOracles() public {
        vm.prank(admin);
        oracle.addOracle(oracle2);

        bytes32 hash1 = keccak256("oracle1 report");
        vm.prank(oracle1);
        oracle.submitReport(token, hash1, 6, "v1.0.0");

        bytes32 hash2 = keccak256("oracle2 report");
        vm.prank(oracle2);
        oracle.submitReport(token, hash2, 4, "v1.1.0");

        // Latest should be from oracle2
        (bytes32 h, uint8 s, , string memory ver) = oracle.latestReport(token);
        assertEq(h, hash2);
        assertEq(s, 4);
        assertEq(ver, "v1.1.0");
        assertEq(oracle.reportCount(token), 2);
    }

    function test_FullWorkflow_ScanAndReport() public {
        vm.prank(user);
        oracle.recordScan(token);
        assertEq(oracle.scanCount(token), 1);

        bytes32 hash = keccak256("scan + report");
        vm.prank(oracle1);
        oracle.submitReport(token, hash, 9, "v1.0.0");

        address user2 = address(0xEEE);
        vm.prank(user2);
        oracle.recordScan(token);

        (uint256 scans, uint256 reports, RiskReportOracle.ReportRecord memory report) =
            oracle.getScanStats(token);

        assertEq(scans, 2);
        assertEq(reports, 1);
        assertEq(report.riskScore, 9);
        assertEq(report.reportHash, hash);
    }

    function test_MultipleTokens_IndependentState() public {
        address tokenA = address(0xAAA1);
        address tokenB = address(0xBBB2);

        bytes32 hashA = keccak256("token A");
        bytes32 hashB = keccak256("token B");

        vm.startPrank(oracle1);
        oracle.submitReport(tokenA, hashA, 3, "v1.0.0");
        oracle.submitReport(tokenB, hashB, 8, "v1.0.0");
        vm.stopPrank();

        // Each token has independent state
        (, uint8 scoreA, , ) = oracle.latestReport(tokenA);
        (, uint8 scoreB, , ) = oracle.latestReport(tokenB);
        assertEq(scoreA, 3);
        assertEq(scoreB, 8);
        assertEq(oracle.reportCount(tokenA), 1);
        assertEq(oracle.reportCount(tokenB), 1);
    }
}
