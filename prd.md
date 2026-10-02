# Product Requirements Document (PRD)

## 1. Product Vision
To provide a secure, fast, and reliable backend API that manages the purchasing and organizational workflows of the company, ensuring strict data integrity, robust access control, and seamless integration with modern frontend applications.

## 2. Target Audience
- **Super Admins:** Full system access, managing roles, regions, and overarching settings.
- **Admins (Staff/Managers):** Segmented by Region and Division. They create and manage POs, PRs, and client relationships based on their specific Role permissions.
- **Approvers:** Specific admins who review and approve Purchase Requisitions.

## 3. Core Features & Requirements

### 3.1 Authentication & Authorization
- **JWT-based Auth:** Access and refresh tokens.
- **Secure Storage:** Redis for refresh-token tracking and access-token blacklisting.
- **Brute Force Protection:** Rate limiting and throttling for login and password change endpoints per IP and account.
- **RBAC:** Every sensitive endpoint must be protected by specific `access_slug` permissions (e.g., `create_po`, `approve_pr`).

### 3.2 Master Data Management
- **Regions & Divisions:** Hierarchical organization structure.
- **Units:** Measurement units for PO items.
- **PPN (Tax):** Manageable tax percentage values applied to transactions.
- **Clients:** Directory of external clients, mapped to regions.

### 3.3 Purchase Orders (PO) Management
- **Creation:** Create POs with header details (Client, Region, Division) and multiple items in a single transaction.
- **Calculation:** Auto-calculate subtotal, PPN, and total based on items.
- **Lifecycle:** Track PO status (`open`, `progress`, `prepared`, `complete`, `cancel`) and important dates.
- **Documents & Activities:** Upload related documents and log activity history on the PO.
- **Export:** Generate detailed Excel (.xlsx) reports of POs with custom template support. Must use caching to optimize repeated requests.

### 3.4 Purchase Requisitions (PR) Management
- **Request Flow:** Admins can request purchases, specifying requested amounts, HPP, and target dates.
- **Approval Workflow:** Multi-level approval system. Approvers can approve, reject, or request revisions. Approvals are tied to digital signatures.
- **Payments:** Track payment stages related to a PR.
- **Linking to PO:** PRs can be linked to existing POs via a Quotation Number bridging mechanism.
- **PR Documents & Comments:** Attach documents to PRs and allow admins to discuss via comments.
- **Cancellation:** Formal cancellation requests that require review.

### 3.5 System & Security
- **No SQL Injection:** Strict use of parameterized SQL queries.
- **Data Integrity:** Use DB transactions for multi-step operations.
- **File Uploads:** Secure handling of images and documents with size and type restrictions.
