# RMS Backend (Ruas Management System)

## General Overview

RMS Backend is a RESTful API service written in Go. It is a complete rewrite and modernization of a legacy PHP monolithic system. The application serves as the core operational backend for managing Purchase Orders (PO), Purchase Requisitions/Requests (PR), Clients, Regional data, and internal administrative workflows.

The frontend is completely decoupled (built with React/Vite) and communicates with this backend purely via JSON REST APIs.

## Key Objectives of the Rewrite

1. **Modernization:** Moving from a legacy PHP architecture to a modern, strongly-typed Go backend.
2. **Security Enhancements:** 
   - Eliminating SQL injection vulnerabilities by using parameterized queries.
   - Replacing basic password hashing with bcrypt.
   - Implementing robust Role-Based Access Control (RBAC) via middleware.
   - Moving from PHP sessions to JWT with Redis-backed token revocation and rate limiting.
3. **Data Integrity:** Replacing loosely coupled manual database updates with strict normalized schema (`rms_normalized.sql`), Foreign Keys with `ON DELETE CASCADE`, and database transactions for complex multi-table inserts (like creating a PO and its items).
4. **Performance:** Utilizing Go's fast HTTP serving capabilities and implementing file-based caching for heavy operations like Excel exports.

## Core Domain Entities

- **Administrative:** Admins (Users), Roles, Access (Permissions).
- **Organization Master Data:** Regions, Divisions, Units, PPN (Tax rates), Responsibles.
- **Clients:** External entities for whom POs are created.
- **Purchase Orders (PO):** The core transactional document, containing multiple items, documents, and activity tracking.
- **Purchase Requisitions (PR):** Internal requests for purchases, featuring a multi-level approval workflow, signature tracking, and payment tracking. PRs can be linked to POs via Quotations.

This project is built for single-instance deployment but is structured cleanly enough to scale if the caching and file-storage mechanisms are adapted for distributed environments.
