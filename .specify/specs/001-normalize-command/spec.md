# Feature Specification: Normalize Command

**Feature Branch**: `001-normalize-command`  
**Created**: 2026-01-08  
**Status**: Draft  
**Input**: User description: "Add a new normalize command to process all JSON files for a specified user and update the normalized_title field by reapplying normalization logic to the track field"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Re-normalize User's Scrobble Files (Priority: P1)

A data administrator needs to update the normalized_title field for all existing scrobble files after improvements or fixes have been made to the normalization logic. They want to ensure all historical data uses the latest normalization rules without having to re-fetch data from Last.fm.

**Why this priority**: Core functionality - enables retroactive application of normalization improvements to existing data, which is the primary purpose of the command.

**Independent Test**: Can be fully tested by running the normalize command on a user's existing files and verifying that normalized_title fields are updated correctly according to current normalization rules, delivering immediate value of consistent data normalization.

**Acceptance Scenarios**:

1. **Given** a user has 100 JSON files with scrobbles in local storage, **When** the administrator runs `./app normalize --user john_doe`, **Then** all 100 files are processed and normalized_title fields are updated based on current normalization rules
2. **Given** a user has scrobble files in Azure Blob Storage, **When** the administrator runs `./app normalize --user jane_doe --azure-account myaccount --azure-container scrobbles`, **Then** all files in Azure storage are processed and updated with new normalized_title values
3. **Given** some files already have correct normalized_title values, **When** the normalize command runs, **Then** only files with changed normalized_title values are updated, unchanged files are left as-is
4. **Given** a file contains scrobbles where track field is "Track #1 - Some Title", **When** normalization is applied, **Then** normalized_title is updated to "track 1 some title" (lowercased, special characters removed)

---

### User Story 2 - Preview Changes Before Applying (Priority: P2)

A data administrator wants to see what changes would be made to normalized_title fields before actually modifying the files, to verify the normalization logic is working as expected and to estimate impact.

**Why this priority**: Important safety feature - allows verification before making bulk changes to data files.

**Independent Test**: Can be fully tested by running normalize command with --dry-run flag and confirming that preview output is shown but no files are modified, providing immediate value of safe verification.

**Acceptance Scenarios**:

1. **Given** a user has 50 files needing normalization updates, **When** the administrator runs `./app normalize --user john_doe --dry-run`, **Then** the system displays which files would be updated showing current and new normalized_title values, but does not write any changes
2. **Given** files are in Azure storage, **When** the administrator runs normalize with --dry-run and Azure flags, **Then** preview is shown without modifying Azure storage
3. **Given** dry-run mode is active, **When** processing completes, **Then** the summary clearly indicates "Dry-run mode: No changes written to storage"

---

### User Story 3 - Monitor Progress and Review Results (Priority: P3)

A data administrator processing hundreds of files wants to see real-time progress during processing and a comprehensive summary afterward to understand what was changed and identify any issues.

**Why this priority**: Enhances user experience - provides visibility and confidence during long-running operations.

**Independent Test**: Can be fully tested by running normalize on a large dataset and verifying progress indicators appear during execution and comprehensive summary is shown at completion.

**Acceptance Scenarios**:

1. **Given** processing 200 files, **When** the normalize command runs, **Then** progress is displayed showing which file is currently being processed
2. **Given** processing completes successfully, **When** the command finishes, **Then** a summary shows total files processed, number updated, number unchanged, and any errors encountered
3. **Given** 5 files fail to parse during processing, **When** the command completes, **Then** the error count is 5 and processing continues for remaining files
4. **Given** processing a mix of files with and without changes, **When** the summary is displayed, **Then** it accurately categorizes files as "updated" or "unchanged"

---

### Edge Cases

- What happens when a file cannot be parsed (malformed JSON)?
- What happens when a file is missing the track field?
- What happens when no files exist for the specified user?
- What happens when normalized_title already matches the newly calculated value?
- What happens when storage permissions prevent reading or writing files?
- What happens when the user specifies both local and Azure flags (conflicting storage targets)?
- What happens when Azure credentials are invalid or the container doesn't exist?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `normalize` command that accepts `--user <username>` as a required argument
- **FR-002**: System MUST support local storage mode when no Azure arguments are provided
- **FR-003**: System MUST support Azure Blob Storage mode when Azure account and container arguments are provided
- **FR-004**: System MUST locate all JSON/NDJSON files for the specified user in the determined storage location
- **FR-005**: System MUST read each file, extract the `track` field, and apply existing normalization logic to generate a new `normalized_title` value
- **FR-006**: System MUST update only the `normalized_title` field in each scrobble record, preserving all other fields unchanged
- **FR-007**: System MUST write updated files back to the same storage location (local or Azure) unless dry-run mode is active
- **FR-008**: System MUST support a `--dry-run` flag that shows what would change without modifying any files
- **FR-009**: System MUST display real-time progress showing which file is currently being processed
- **FR-010**: System MUST generate a summary report showing total files processed, number updated, number unchanged, and error count
- **FR-011**: System MUST continue processing remaining files when individual files fail to parse or process
- **FR-012**: System MUST report all errors encountered during processing in the summary
- **FR-013**: System MUST clearly indicate in output when dry-run mode is active and no changes are written
- **FR-014**: System MUST use the same Azure configuration pattern and argument names as existing fetch and merge commands
- **FR-015**: System MUST use the same storage abstraction layer as existing commands for consistency
- **FR-016**: System MUST handle files that already have correct normalized_title values by skipping updates for those files
- **FR-017**: System MUST display both current and new normalized_title values during dry-run mode for files that would change
- **FR-018**: System MUST validate that required user argument is provided and error appropriately if missing
- **FR-019**: System MUST validate Azure configuration when Azure mode is used and error appropriately if incomplete or invalid

### Key Entities

- **Scrobble File**: Represents a JSON/NDJSON file containing scrobble records for a user, stored in either local filesystem or Azure Blob Storage
- **Scrobble Record**: Individual listening event containing fields including track (original title) and normalized_title (processed title)
- **Storage Location**: Either local filesystem or Azure Blob Storage container, determined by command-line arguments provided

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Administrator can process all files for a user in under 5 seconds per 1000 files
- **SC-002**: System correctly identifies and updates 100% of files where normalized_title differs from newly calculated value
- **SC-003**: Zero data loss - all fields except normalized_title remain unchanged after processing
- **SC-004**: Dry-run mode produces accurate preview - 100% match between preview and actual changes when run without --dry-run
- **SC-005**: System continues processing and completes successfully even when up to 10% of files encounter parsing errors
- **SC-006**: Summary report provides complete accounting - sum of updated, unchanged, and error counts equals total files processed
