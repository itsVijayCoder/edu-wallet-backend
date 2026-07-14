# Guardian and Parent API rollout

This rollout fixes the guardian list/detail contract, the unified parents list,
and production databases whose guardian-user schema drifted from the recorded
migration version.

## 1. Before deployment

1. Take a database backup or snapshot.
2. Confirm the application is using the intended database URL.
3. Check the current migration state:

   ```sh
   make migrate-version
   ```

   Do not deploy against a dirty migration state. Investigate the failed SQL
   before using `migrate-force`; forcing a version does not repair schema.

## 2. Apply the migration

Run:

```sh
make migrate-up
make migrate-version
```

The expected clean version is `12`. Migration `000012` is idempotent for the
objects involved in this incident. It:

- repairs `guardians.user_id` and its foreign key/indexes if missing;
- adds and backfills `guardians.opt_in_whatsapp`;
- repairs the `student_guardians` table and indexes if the table is missing.

The Docker image runs `./eduwallet-migrate up` from `render-start.sh` before
starting the API. On Render, verify the service command remains
`./render-start.sh` and confirm version 12 in the deployment logs.

## 3. Verify the schema

Run these read-only checks with your PostgreSQL client:

```sql
SELECT version, dirty FROM schema_migrations;

SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'guardians'
  AND column_name IN ('user_id', 'opt_in_whatsapp')
ORDER BY column_name;

SELECT to_regclass('public.student_guardians') AS student_guardians_table;
```

Expected results: version `12`, `dirty = false`, both guardian columns are
present, and `student_guardians_table` is `student_guardians`.

## 4. Smoke test

Using an admin tenant access token, verify:

```sh
curl -fsS "$BASE_URL/api/v1/admin/guardians?page=1&page_size=20" \
  -H "Authorization: Bearer $TENANT_TOKEN"

curl -fsS "$BASE_URL/api/v1/admin/parents?page=1&page_size=20" \
  -H "Authorization: Bearer $TENANT_TOKEN"
```

Both requests must return HTTP 200 with the standard `data` and `meta`
envelope. Linked rows must contain `user_id` and `user_status`; parent rows
must always contain `linked_students` as an array (possibly empty).

Also verify link/unlink and reverse lookup for one known guardian:

```sh
curl -fsS -X POST "$BASE_URL/api/v1/admin/guardians/$GUARDIAN_ID/user" \
  -H "Authorization: Bearer $TENANT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$PARENT_USER_ID\"}"

curl -fsS "$BASE_URL/api/v1/admin/guardians/$GUARDIAN_ID/students" \
  -H "Authorization: Bearer $TENANT_TOKEN"

curl -fsS -X DELETE "$BASE_URL/api/v1/admin/guardians/$GUARDIAN_ID/user" \
  -H "Authorization: Bearer $TENANT_TOKEN"
```

The linked user must exist and hold the `parents` role.

## 5. Rollback

Rollback only if the new application has not started writing the new field:

```sh
make migrate-down n=1
```

The down migration removes only `opt_in_whatsapp`. It deliberately preserves
the older `user_id` link and `student_guardians` data owned by migrations 11
and 5.
