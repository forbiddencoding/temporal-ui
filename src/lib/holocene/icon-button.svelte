<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';

  import { twMerge as merge } from 'tailwind-merge';

  import type { IconName } from '$lib/holocene/icon';
  import Icon from '$lib/holocene/icon/icon.svelte';

  interface $$Props extends HTMLButtonAttributes {
    icon: IconName;
    'data-testid'?: string;
    label: string;
  }

  let className = '';
  export { className as class };
  export let icon: IconName;
  export let label: string;
</script>

<button
  type="button"
  class={merge(
    'inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 border-[transparent] text-primary outline-none hover:surface-interactive-secondary focus-visible:surface-interactive-secondary focus-visible:border-indigo-600 disabled:cursor-not-allowed disabled:opacity-50',
    className,
  )}
  on:click|stopPropagation
  aria-label={label}
  {...$$restProps}
>
  <Icon name={icon} />
</button>
