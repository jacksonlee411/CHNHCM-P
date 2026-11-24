import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectDetail } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { country } from './countryAdapter';
export { COUNTRY_PATH } from './countryAdapter';
export function CountryList() { return <ObjectDefinition value={country}><ObjectList /></ObjectDefinition>; }
export function CountryDetail({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={country}><ObjectDetail session={session} /></ObjectDefinition>; }
export function CountryRecovery({ session }: { session: Models['Principal'] }) { return <SubmissionRecovery definition={country} session={session} />; }
