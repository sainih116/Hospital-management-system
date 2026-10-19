import Autocomplete from "@mui/material/Autocomplete";
import TextField from "@mui/material/TextField";
import { useEffect, useState } from "react";

export interface AsyncOption {
  id: string;
  label: string;
}

function useDebouncedValue(value: string, delayMs: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}

interface AsyncAutocompleteProps {
  label: string;
  value: AsyncOption | null;
  onChange: (option: AsyncOption | null) => void;
  fetchOptions: (query: string) => Promise<AsyncOption[]>;
  helperText?: string;
  error?: boolean;
  required?: boolean;
}

/** A search-as-you-type Autocomplete backed by a server-side search call,
 * used anywhere the app needs to pick a patient/doctor by name. */
export default function AsyncAutocomplete({ label, value, onChange, fetchOptions, helperText, error, required }: AsyncAutocompleteProps) {
  const [inputValue, setInputValue] = useState("");
  const [options, setOptions] = useState<AsyncOption[]>([]);
  const [loading, setLoading] = useState(false);
  const debouncedInput = useDebouncedValue(inputValue, 300);

  useEffect(() => {
    if (!debouncedInput) {
      setOptions([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    fetchOptions(debouncedInput)
      .then((results) => {
        if (!cancelled) setOptions(results);
      })
      .catch(() => {
        if (!cancelled) setOptions([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedInput]);

  return (
    <Autocomplete
      options={options}
      loading={loading}
      filterOptions={(x) => x}
      value={value}
      onInputChange={(_, newValue) => setInputValue(newValue)}
      onChange={(_, option) => onChange(option)}
      isOptionEqualToValue={(option, val) => option.id === val.id}
      renderInput={(params) => <TextField {...params} label={label} required={required} error={error} helperText={helperText} />}
    />
  );
}
