# Reference mapping runtime design

MongoDB is the sole source of truth for campuses, departments, and course
categories. Redis accelerates reads but is never authoritative. Compiled Go maps
remain only as migration seeds and as dependency-free unit-test fixtures.

Read flow:

1. Read the appropriate Redis Hash.
2. On a miss or Redis failure, query MongoDB.
3. Refill both Redis directions after a successful MongoDB lookup.

Write flow:

1. Allocate a code through `mapping_counter` and write MongoDB.
2. Commit the surrounding MongoDB transaction when one exists.
3. Update or rebuild Redis only after commit.

Campus names must already exist. Proposal creation and approval may select
multiple known campuses but cannot register a new campus. Departments and course
categories may be created while approving a proposal.

Bulk course assembly uses `HMGET` for each mapping type. Startup loads all mapping
documents, constructs temporary bidirectional hashes, and switches the six live
keys in one Lua operation. Concurrent instances therefore observe either the old
complete snapshot or the new complete snapshot.

Some historical static lists contain the same display name under multiple codes.
All codes remain valid for `code -> name`; exactly one record is marked
`canonical` for `name -> code`. The migration chooses the lowest historical
static code as canonical and never treats an online dynamic/static collision as
an alias.
