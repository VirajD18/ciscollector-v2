package userlist

// top25Runner holds Users Report queries aligned with Top 25 critical checks.
var top25Runner = []UserlistHelper{
	{
		Title: "SCRAM-SHA-256 password_encryption",
		Note:  "Top 25 check #1 — password_encryption must be scram-sha-256",
		Query: `SELECT current_setting('password_encryption') AS password_encryption,
       (current_setting('password_encryption') = 'scram-sha-256') AS pass;`,
	},
	{
		Title: "Roles with non-SCRAM password hashes",
		Note:  "Top 25 check #2 — MD5 or legacy password hashes should be rotated to SCRAM-SHA-256",
		Query: `SELECT count(*)::text AS non_scram_count,
       (count(*) = 0) AS pass
       FROM pg_authid
       WHERE rolpassword IS NOT NULL AND rolpassword NOT LIKE 'SCRAM-SHA-256%';`,
	},
	{
		Title: "Superuser count (Top 25 check #7)",
		Note:  "At most 3 human superuser roles (excluding pg_* system roles)",
		Query: `SELECT count(*)::text AS superuser_count,
       (count(*) <= 3) AS pass
       FROM pg_roles
       WHERE rolsuper IS TRUE AND rolname NOT LIKE 'pg_%';`,
	},
	{
		Title: "listen_addresses not wildcard (Top 25 check #9)",
		Note:  "listen_addresses must not be *",
		Query: `SELECT current_setting('listen_addresses') AS listen_addresses,
       (current_setting('listen_addresses') <> '*') AS pass;`,
	},
	{
		Title: "log_connections enabled (Top 25 check #11)",
		Query: `SELECT current_setting('log_connections') AS setting,
       (current_setting('log_connections') IN ('on', 'all')) AS pass;`,
	},
	{
		Title: "log_disconnections enabled (Top 25 check #12)",
		Query: `SELECT current_setting('log_disconnections') AS setting,
       (current_setting('log_disconnections') = 'on') AS pass;`,
	},
	{
		Title: "log_statement at least ddl (Top 25 check #13)",
		Query: `SELECT current_setting('log_statement') AS setting,
       (current_setting('log_statement') IN ('ddl', 'mod', 'all')) AS pass;`,
	},
	{
		Title: "log_line_prefix format (Top 25 check #14)",
		Query: `SELECT current_setting('log_line_prefix') AS log_line_prefix,
       (
         current_setting('log_line_prefix') LIKE '%\%m%' AND
         current_setting('log_line_prefix') LIKE '%\%p%' AND
         current_setting('log_line_prefix') LIKE '%\%l%' AND
         current_setting('log_line_prefix') LIKE '%\%d%' AND
         current_setting('log_line_prefix') LIKE '%\%u%' AND
         current_setting('log_line_prefix') LIKE '%\%a%' AND
         current_setting('log_line_prefix') LIKE '%\%h%'
       ) AS pass;`,
	},
	{
		Title: "log_destination persistent (Top 25 check #15)",
		Query: `SELECT current_setting('log_destination') AS log_destination,
       current_setting('logging_collector') AS logging_collector,
       (
         current_setting('logging_collector') = 'on' OR
         current_setting('log_destination') LIKE '%syslog%'
       ) AS pass;`,
	},
	{
		Title: "pgaudit extension installed (Top 25 check #16)",
		Query: `SELECT current_setting('shared_preload_libraries') AS shared_preload_libraries,
       (current_setting('shared_preload_libraries') LIKE '%pgaudit%') AS pass;`,
	},
	{
		Title: "pgaudit.log configured (Top 25 check #17)",
		Query: `SELECT coalesce(current_setting('pgaudit.log', true), '') AS pgaudit_log,
       (
         lower(coalesce(current_setting('pgaudit.log', true), '')) LIKE '%role%' AND
         lower(coalesce(current_setting('pgaudit.log', true), '')) LIKE '%ddl%' AND
         lower(coalesce(current_setting('pgaudit.log', true), '')) LIKE '%write%'
       ) AS pass;`,
	},
	{
		Title: "SECURITY DEFINER functions (Top 25 check #6)",
		Note:  "Review SECURITY DEFINER functions — zero is a pass for this check",
		Query: `SELECT count(*)::text AS secdef_count,
       (count(*) = 0) AS pass
       FROM pg_proc p
       JOIN pg_namespace n ON p.pronamespace = n.oid
       WHERE p.prosecdef
         AND n.nspname NOT IN ('pg_catalog', 'information_schema');`,
	},
	{
		Title: "Databases open to PUBLIC connect",
		Note:  "Top 25 check #24 — REVOKE CONNECT FROM PUBLIC on sensitive databases",
		Query: `SELECT datname FROM pg_database WHERE has_database_privilege('PUBLIC', datname, 'CONNECT');`,
	},
}
