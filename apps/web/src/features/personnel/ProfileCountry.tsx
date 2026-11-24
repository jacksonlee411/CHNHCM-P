import { useEffect, useState } from 'react';
import { Autocomplete, TextField } from '@mui/material';
import { ObjectDetailField } from '@chnhcm/ui';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { message } from './api';

type Option = { id: string; name: string; path?: string; choice: Choice };
export type Choice = Models['PersonPlaceChoice'];
export function ProfileCountry({
  id,
  label,
  value,
  disabled,
  required,
  fieldError,
  region,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  disabled?: boolean;
  required?: boolean;
  fieldError?: string;
  region?: { country?: string; parent?: string; province: boolean };
  onChange: (text: string, choice?: Choice) => void;
}) {
  const [open, setOpen] = useState(false),
    [options, setOptions] = useState<Option[]>([]),
    [error, setError] = useState('');
  const [query, setQuery] = useState(''),
    [epoch, setEpoch] = useState(0);
  useEffect(() => {
    if (!open) return;
    const refresh = () => {
      setOptions([]);
      setEpoch((v) => v + 1);
    };
    const visible = () => {
      if (document.visibilityState === 'visible') refresh();
    };
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', visible);
    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', visible);
    };
  }, [open]);
  useEffect(() => {
    if (!open || (region && (!region.country || (!region.province && !region.parent)))) {
      setOptions([]);
      return;
    }
    const controller = new AbortController(),
      current = sessionGuard();
    setOptions([]);
    const timer = setTimeout(async () => {
      try {
        if (region?.country) {
          const { currentDate } = unwrap(
            await api.GET('/api/v1/geo-areas/context', { signal: controller.signal }),
          );
          const result = unwrap(
            await api.GET('/api/v1/geo-areas/candidates', {
              params: {
                query: {
                  countryRegionId: region.country,
                  parentAreaId: region.parent,
                  adoptedDate: currentDate,
                  q: query || undefined,
                  pageSize: 100,
                },
              },
              signal: controller.signal,
            }),
          );
          if (current() && !controller.signal.aborted) {
            setOptions(
              result.items.flatMap((item) => {
                const leaf = item.path.at(-1);
                if (
                  !leaf?.sourceVersion ||
                  (region.province
                    ? item.path.length !== 2
                    : item.path.length < 3 || item.path[1].id !== region.parent)
                )
                  return [];
                return [
                  {
                    id: item.id,
                    name: item.name,
                    path: item.path.map((node) => node.name).join(' / '),
                    choice: {
                      id: item.id,
                      sourceVersion: leaf.sourceVersion,
                      ancestors: item.path
                        .slice(0, -1)
                        .map((x) => ({ id: x.id, sourceVersion: x.sourceVersion! })),
                    },
                  },
                ];
              }),
            );
            setError('');
          }
          return;
        }
        const list = unwrap(
          await api.GET('/api/v1/country-regions', {
            params: { query: { q: query || undefined, status: 'active', page: 1, pageSize: 20 } },
            signal: controller.signal,
          }),
        );
        if (current() && !controller.signal.aborted) {
          setOptions(
            list.items.flatMap((x) =>
              x.name
                ? [{ id: x.id, name: x.name, choice: { id: x.id, sourceVersion: x.sourceVersion } }]
                : [],
            ),
          );
          setError('');
        }
      } catch (e) {
        if (current() && !controller.signal.aborted) {
          setOptions([]);
          setError(message(e));
        }
      }
    }, 180);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [open, query, epoch, region?.country, region?.parent, region?.province]);
  return (
    <ObjectDetailField label={label} inputId={id} required={required}>
      <Autocomplete
        freeSolo
        id={id}
        disabled={disabled}
        options={options}
        open={open}
        onOpen={() => setOpen(true)}
        onClose={() => {
          setOpen(false);
          setQuery('');
        }}
        inputValue={value}
        getOptionKey={(option) => (typeof option === 'string' ? option : option.id)}
        renderOption={(props, option) => {
          const { key, ...attributes } = props;
          const duplicate = options.some((other) => other.id !== option.id && other.name === option.name);
          return (
            <li key={key} {...attributes}>
              {duplicate && option.path ? option.path : option.name}
            </li>
          );
        }}
        getOptionLabel={(o) => (typeof o === 'string' ? o : o.name)}
        filterOptions={(o) => o}
        onInputChange={(_, v, reason) => {
          if (reason === 'input' || reason === 'clear') {
            setError('');
            setQuery(v);
            onChange(v);
          }
        }}
        onChange={(_, v) => {
          if (v && typeof v !== 'string') {
            onChange(v.name, v.choice);
            setError('');
          }
        }}
        renderInput={(params) => (
          <TextField
            {...params}
            variant="standard"
            required={required}
            error={!!(error || fieldError)}
            helperText={error || fieldError}
          />
        )}
      />
    </ObjectDetailField>
  );
}
