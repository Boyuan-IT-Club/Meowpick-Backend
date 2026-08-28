# User profile and proposal consistency

This document records the runtime behavior and production rollout checks for
the user-profile and proposal fixes. MongoDB remains the source of truth. Redis
and the `monc` object cache are disposable read caches.

## Corrected API behavior

- `GET /api/proposal/list` now binds `status` from the query string. Ordinary
  users always receive approved proposals; administrators may request one
  status or omit it to list all statuses.
- `GET /api/proposal/filter` accepts omitted `status` and `campus` parameters.
  Ordinary users are still forced to approved data. Campus validation uses the
  runtime MongoDB/Redis mapping rather than the old compiled mapping table.
- Proposal create, update, and approval reject blank titles, course names,
  departments, categories, empty campus lists, malformed teachers, and unknown
  campuses. Approval validates the administrator's final course again.
- `POST /api/proposal/:proposalId/delete` accepts an empty body because the
  proposal ID is supplied by the path.
- `GET /api/user/:userId/username` allows self lookup and administrator lookup.
  An ordinary cross-user lookup must supply `proposalId`, and that proposal must
  belong to the target user and have `showUsername=true`.

The complete affected route set is:

- `GET /api/user/profile`, `POST /api/user/profile/update`, and
  `GET /api/user/:userId/username`;
- `POST /api/proposal/add`, `GET /api/proposal/list`,
  `GET /api/proposal/filter`, `POST /api/proposal/:proposalId/update`, and
  `POST /api/proposal/:proposalId/delete`;
- `POST /api/proposal/:proposalId/approve`,
  `POST /api/proposal/:proposalId/reject`, and
  `POST /api/proposal/:proposalId/revoke`.

## Consistency guarantees

- Approval and revoke-approval execute course, mapping, teacher, contribution,
  proposal-status, and audit-log writes in a MongoDB transaction. A concurrent
  second revoke cannot debit contribution twice, and contribution cannot become
  negative.
- Update, delete, approve, reject, and revoke use expected-status predicates.
  If another request wins the state transition, the loser returns a proposal
  update failure instead of applying stale work.
- Proposal creation uses two `proposal_guard` documents: one for the user and
  UTC+8 calendar day, and one for the normalized course fingerprint. Updating
  those documents inside the transaction serializes the daily-quota and
  duplicate checks. Guard rows contain only IDs, a version, and timestamps.
- Nickname updates include the previously read `usernameUpdatedAt` in the MongoDB
  predicate. Concurrent requests therefore cannot both pass the 30-day check.
- User reads inside MongoDB transactions bypass `monc`. Contribution writes
  invalidate the user object cache only after a successful commit, so
  `/api/user/profile` immediately observes approval or revoke results.

## Database changes

No existing business collection is replaced and no field is renamed. The
release creates or uses:

- `user.idx_user_username_unique`: case-insensitive, unique, partial index for
  non-empty `username` values;
- `proposal_guard`: new non-authoritative coordination collection using the
  default unique `_id` index. A TTL index removes inactive guard rows after 48
  hours; deletion is safe because every transaction still checks MongoDB
  business collections as the source of truth.

Run `cmd/migrate-v2` in dry-run mode before deployment. It now stops and writes
the conflict report if legacy nicknames collide case-insensitively or a user's
contribution is already negative. Do not start the new backend until all
conflicts are resolved and a fresh dry-run is clean. Follow
`docs/MIGRATION-V2.md` for backup, apply, validation, and rollback commands.

## Deployment verification

1. Back up MongoDB and run the mandatory migration dry-run.
2. Apply the exact unchanged plan during the full application write-maintenance
   window required by `docs/MIGRATION-V2.md`.
3. Start the backend and verify MongoDB is a replica set or sharded deployment.
4. Create a pending proposal with `POST /api/proposal/add`. Then call both
   `GET /api/proposal/list` and `GET /api/proposal/filter` without `status` or
   `campus`; an ordinary user must not receive that pending proposal.
5. Approve a test proposal, then immediately read `/api/user/profile`; the new
   contribution must be visible without restarting or flushing Redis.
6. Send two concurrent revoke requests. Exactly one may succeed; course,
   proposal contribution, user contribution, and proposal status must agree.
7. Verify a hidden-name proposal cannot be used by another ordinary user to
   resolve its author's nickname.

## Rollback

Application rollback does not require deleting `proposal_guard`; old versions
ignore it. If the database migration itself must be rolled back, restore the
verified pre-migration backup as documented in `docs/MIGRATION-V2.md`, flush only
the application's Redis database, and start the previous backend. Never repair
production conflicts by overwriting dynamic mapping codes or usernames without
an operator-reviewed conflict resolution.
