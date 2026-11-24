// Compile-only port checks. This synthetic object is not a Department implementation.
import type { DatedObject, FieldCatalog, ResultView } from '../src/features/organization/dated/object';
import type { RecordCommand } from '../src/features/organization/businessUnitAdapter';
import { submissionChannel } from '../src/shared/dated/submission';

type SampleResult = { submissionId: string; outcome: 'accepted'; sampleId: string; source: string };
export const sample: DatedObject<'fixture.sample', SampleResult> = {
  kind: 'fixture.sample', basePath: '/fixture/sample', submissions: submissionChannel('fixture.sample', 'fixture-sample'),
  resolve: async () => { throw new Error('compile only'); },
  resultView: value => ({ kind: 'fixture.sample', submissionId: value.submissionId, outcome: value.outcome, objectId: value.sampleId, sourceVersion: value.source, lines: [] }),
};
const fields = { title: { label: 'Title' }, reference: { label: 'Reference' } } satisfies FieldCatalog<{ title: string; reference?: string }>;
void fields;
function rejectWrongMappings(result: SampleResult, command: RecordCommand, view: ResultView<'fixture.sample'>) {
  // @ts-expect-error sample result has no BU-specific field.
  void result.buId;
  // @ts-expect-error object route/result kinds are not interchangeable.
  const wrong: ResultView<'organization.business-unit'> = view;
  // @ts-expect-error a single maintenance command is not a CreateInput.
  const create: import('@chnhcm/api-client').Models['CreateInput'] = command;
  void wrong; void create;
}
void rejectWrongMappings;
