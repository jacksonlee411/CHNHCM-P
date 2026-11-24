import type { Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { ObjectList, ObjectDetail, ObjectNew } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { location } from './locationAdapter';
export { LOCATION_PATH } from './locationAdapter';
export function LocationList() { return <ObjectDefinition value={location}><ObjectList /></ObjectDefinition>; }
export function LocationDetail({ session }: {
    session: Models['Principal'];
}) { return <ObjectDefinition value={location}><ObjectDetail session={session}/></ObjectDefinition>; }
export function LocationRecovery({ session }: {
    session: Models['Principal'];
}) { return <SubmissionRecovery definition={location} session={session}/>; }
export function NewLocation({ session }: {
    session: Models['Principal'];
}) { return <ObjectDefinition value={location}><ObjectNew session={session}/></ObjectDefinition>; }
