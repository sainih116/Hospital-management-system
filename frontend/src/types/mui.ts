import type SvgIcon from "@mui/material/SvgIcon";

/**
 * `@mui/material` (installed: 6.5.0) has never re-exported `SvgIconComponent`
 * from its package root — the name only ever existed as an unexported,
 * file-local alias inside `@mui/icons-material`'s own generated index
 * (`type SvgIconComponent = typeof SvgIcon`), which is how every generated
 * icon (e.g. `PeopleAltRoundedIcon`) is typed there. Reconstructed here,
 * identically, so call sites that need to type "a component from
 * @mui/icons-material" have one shared, correctly-typed alias instead of
 * each importing a member that was never actually exported.
 */
export type SvgIconComponent = typeof SvgIcon;
