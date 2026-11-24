/** Applies the server-provided default without inferring business eligibility. */
export function selectedReason(previous: string, choices: ReadonlyArray<{ code: string; default: boolean }>): string {
  return choices.some(choice => choice.code === previous) ? previous : choices.find(choice => choice.default)?.code ?? '';
}
