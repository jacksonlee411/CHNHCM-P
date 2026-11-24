import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectDetail } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { geoArea } from './geoAreaAdapter';
export { GEOAREA_PATH } from './geoAreaAdapter';
export function GeoAreaList() { return <ObjectDefinition value={geoArea}><ObjectList /></ObjectDefinition>; }
export function GeoAreaDetail({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={geoArea}><ObjectDetail session={session} /></ObjectDefinition>; }
export function GeoAreaRecovery({ session }: { session: Models['Principal'] }) { return <SubmissionRecovery definition={geoArea} session={session} />; }
