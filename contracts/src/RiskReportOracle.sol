// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/// @title RiskReportOracle
/// @notice Anchors Meme token risk report hashes on-chain for immutable audit trail
/// @dev Self-contained implementation with inline access control (no OpenZeppelin dependency)
contract RiskReportOracle {
    // ========== Errors ==========

    error Unauthorized(address caller, bytes32 requiredRole);
    error InvalidTokenAddress();
    error EmptyReportHash();
    error InvalidRiskScore(uint8 score);
    error EmptyVersion();
    error InvalidOracleAddress();

    // ========== Constants ==========

    bytes32 public constant DEFAULT_ADMIN_ROLE = 0x00;
    bytes32 public constant ORACLE_ROLE = keccak256("ORACLE_ROLE");

    // ========== Structs ==========

    struct ReportRecord {
        bytes32 reportHash; // keccak256 of the full report (stored on Arweave/IPFS)
        uint8 riskScore; // 1-10, higher = more dangerous
        uint256 timestamp; // block timestamp of submission
        string version; // analysis engine version (e.g. "v1.2.0")
    }

    // ========== State ==========

    /// @notice role => account => hasRole
    mapping(bytes32 => mapping(address => bool)) private _roles;

    /// @notice tokenAddress => latest report
    mapping(address => ReportRecord) public latestReport;

    /// @notice tokenAddress => total number of reports submitted
    mapping(address => uint256) public reportCount;

    /// @notice tokenAddress => total number of scans performed
    mapping(address => uint256) public scanCount;

    // ========== Events ==========

    event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender);
    event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender);

    event ReportSubmitted(
        address indexed token,
        bytes32 indexed reportHash,
        uint8 riskScore,
        uint256 timestamp,
        uint256 totalReports
    );

    event TokenScanned(address indexed token, address indexed scannedBy, uint256 totalScans);

    // ========== Modifiers ==========

    modifier onlyRole(bytes32 role) {
        _checkRole(role, msg.sender);
        _;
    }

    // ========== Constructor ==========

    constructor() {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(ORACLE_ROLE, msg.sender);
    }

    // ========== Access Control ==========

    /// @notice Check if an account has a role
    function hasRole(bytes32 role, address account) public view returns (bool) {
        return _roles[role][account];
    }

    /// @notice Grant a role to an account (admin only)
    function grantRole(bytes32 role, address account) public onlyRole(DEFAULT_ADMIN_ROLE) {
        _grantRole(role, account);
    }

    /// @notice Revoke a role from an account (admin only)
    function revokeRole(bytes32 role, address account) public onlyRole(DEFAULT_ADMIN_ROLE) {
        _revokeRole(role, account);
    }

    /// @notice Renounce a role from the caller
    function renounceRole(bytes32 role, address account) public {
        require(account == msg.sender, "can only renounce roles for self");
        _revokeRole(role, account);
    }

    function _checkRole(bytes32 role, address account) internal view {
        if (!hasRole(role, account)) {
            revert Unauthorized(account, role);
        }
    }

    function _grantRole(bytes32 role, address account) internal {
        if (!hasRole(role, account)) {
            _roles[role][account] = true;
            emit RoleGranted(role, account, msg.sender);
        }
    }

    function _revokeRole(bytes32 role, address account) internal {
        if (hasRole(role, account)) {
            _roles[role][account] = false;
            emit RoleRevoked(role, account, msg.sender);
        }
    }

    // ========== Write Functions ==========

    /// @notice Submit a risk report (ORACLE_ROLE only)
    /// @param token The meme token address being analyzed
    /// @param reportHash keccak256 hash of the full report data
    /// @param riskScore 1-10 risk score
    /// @param version Analysis engine version
    function submitReport(
        address token,
        bytes32 reportHash,
        uint8 riskScore,
        string calldata version
    ) external onlyRole(ORACLE_ROLE) {
        if (token == address(0)) revert InvalidTokenAddress();
        if (reportHash == bytes32(0)) revert EmptyReportHash();
        if (riskScore < 1 || riskScore > 10) revert InvalidRiskScore(riskScore);
        if (bytes(version).length == 0) revert EmptyVersion();

        latestReport[token] = ReportRecord({
            reportHash: reportHash,
            riskScore: riskScore,
            timestamp: block.timestamp,
            version: version
        });
        reportCount[token]++;

        emit ReportSubmitted(token, reportHash, riskScore, block.timestamp, reportCount[token]);
    }

    /// @notice Record a scan event (permissionless, triggered by frontend on query)
    /// @param token The token address being scanned
    function recordScan(address token) external {
        if (token == address(0)) revert InvalidTokenAddress();
        scanCount[token]++;
        emit TokenScanned(token, msg.sender, scanCount[token]);
    }

    // ========== Read Functions ==========

    /// @notice Get the latest report for a token
    function getLatestReport(address token) external view returns (ReportRecord memory) {
        return latestReport[token];
    }

    /// @notice Get the total number of reports for a token
    function getReportCount(address token) external view returns (uint256) {
        return reportCount[token];
    }

    /// @notice Get combined scan statistics for a token
    function getScanStats(address token)
        external
        view
        returns (uint256 scans, uint256 reports, ReportRecord memory lastReport)
    {
        return (scanCount[token], reportCount[token], latestReport[token]);
    }

    // ========== Admin Functions ==========

    /// @notice Add a new oracle address
    function addOracle(address oracle) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (oracle == address(0)) revert InvalidOracleAddress();
        grantRole(ORACLE_ROLE, oracle);
    }

    /// @notice Remove an oracle address
    function removeOracle(address oracle) external onlyRole(DEFAULT_ADMIN_ROLE) {
        revokeRole(ORACLE_ROLE, oracle);
    }
}
