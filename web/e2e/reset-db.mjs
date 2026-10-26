// Remet à zéro la base jetable des tests e2e (aucun psql requis sur la machine).
import pg from 'pg'

const url = process.env.E2E_DATABASE_URL
if (!url) {
  console.error('E2E_DATABASE_URL manquant : base jetable, remise à zéro à chaque passage')
  process.exit(1)
}
const client = new pg.Client({ connectionString: url })
await client.connect()
await client.query('DROP SCHEMA public CASCADE; CREATE SCHEMA public;')
await client.end()
console.log('base e2e remise à zéro')
