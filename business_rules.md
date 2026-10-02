# Business Rules

This document outlines the core business rules and logic implemented in the RMS Backend.

## 1. Purchase Request (PR)

### 1.1 PR Submission Rules
- **Status Requirements:** A PR must be in the `draft` or `revision` state to be submitted.
- **Checker Assignment:** The requester must assign a valid `checker` (Level 1 Approver). The requester **cannot** assign themselves as the checker.
- **Cost Control Limit:** If the PR's `RequestedAmount` is greater than **Rp 50,000,000**, a "Cost Control" document must be attached to the PR before it can be submitted.
- **Digital Signatures:** The requester must have an active digital signature registered in their profile to submit a PR.

### 1.2 PR Approval Workflow
The approval process consists of 3 sequential levels per round:
1. **Level 1 (Checker):** The specifically assigned admin.
2. **Level 2 (Director):** Any admin possessing the `director` access slug.
3. **Level 3 (Finance):** Any admin possessing the `finance` access slug.

**Rules:**
- **Sequential Approval:** Approvals must happen in order. Level 2 cannot approve until Level 1 has approved, and Level 3 cannot approve until Level 2 has approved.
- **Self-Approval:** A requester cannot approve their own PR, even if they hold the appropriate access level (e.g., a Director cannot approve their own PR at Level 2).
- **Approver Signatures:** Every approver must have a digital signature uploaded; it is attached to the approval record upon decision.
- **Rejection & Revision:** 
  - Only **Level 1 (Checker)** is allowed to `reject` a PR or request a `revision`.
  - Rejecting transitions the PR to the `rejected` state.
  - Requesting a revision transitions the PR to the `revision` state.
- **Completion:** Once Level 3 (Finance) approves, the PR officially transitions to the `approved` state, and draft payments are activated.

### 1.3 PR to PO Linking (Quotations)
- PRs are grouped together by a shared `Quotation No` (qout_no).
- If a PR references a previous PR to form a chain, they must share the same `Responsible` (Chart of Accounts) and `Quotation No`. Circular references are prohibited.

## 2. Purchase Orders (PO)

### 2.1 Creation and Management
- **Client Handling:** If an existing `ClientID` is provided, the client must be active. If no `ClientID` is provided but complete client details (name, email, phone, address) are given, a new Client is implicitly created.
- **Total Calculations:** Adding, updating, or deleting items in a PO automatically triggers a recalculation of the PO's subtotal, PPN (tax) amount, and total cost based on the configured PPN rate.
- **Unique Order Numbers:** PO Order Numbers must be unique across the system.

### 2.2 PO and PR Linking
- **Linkable States:** A PO can only be linked to a Quotation (PR Group) if the PO is in the `prepared`, `progress`, or `complete` state.
- **Quotation Eligibility:** A Quotation can only be linked to a PO if at least one PR within that Quotation group has reached the `completed` state.
- **One-to-One Link:** A Quotation can only be linked to one PO at a time.

### 2.3 Status Changes and Deletion
- **Cancellation Restriction:** A PO **cannot** be changed to `cancel` or reverted to `open` if it has linked Quotations. The user must manually unlink the Quotations first.
- **Deletion Cascade:** Deleting a PO will automatically unlink any associated Quotations and leave an audit note on the PR comments detailing the deletion.

## 3. Caching and Exports

### 3.1 PO Excel Export
- **File-based Caching:** Generating Excel exports is a heavy operation. The system hashes the requested filters (date ranges, region, status, etc.) to create a cache key.
- **Cache Hits:** If an identical request is made within the TTL (default 20 minutes), the backend serves the physical file from the disk cache (`HIT`) without querying the database, drastically improving performance.

## 4. Security & Authentication
- **Rate Limiting:** Specific rate limits and throttles apply to login attempts and password change requests to prevent brute-force attacks. 
- **Token Invalidation:** Upon logout or password change, the current JWT access token is placed in a Redis blacklist, ensuring immediate revocation. Refresh tokens are strictly rotated.
