<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';
  import LabeledSelect from './LabeledSelect.svelte';

  interface InheritedCfg {
    desktop_width?: string;
    desktop_height?: string;
    desktop_color_depth?: string;
  }

  interface Props {
    width: string;
    height: string;
    colorDepth: string; // '8' | '16' | '24' | '32'
    inherited?: InheritedCfg;
  }

  let {
    width = $bindable(''),
    height = $bindable(''),
    colorDepth = $bindable('32'),
    inherited
  }: Props = $props();

  const depthOptions = [
    { value: '8', label: '8-bit' },
    { value: '16', label: '16-bit' },
    { value: '24', label: '24-bit' },
    { value: '32', label: '32-bit' }
  ];
</script>

<div class="grid grid-cols-3 gap-3">
  <LabeledInput id="desktop_width" label="Width" bind:value={width} placeholder={inherited?.desktop_width ? `Inherited: ${inherited.desktop_width}` : '1920'} inherited={inherited?.desktop_width || ''} />
  <LabeledInput id="desktop_height" label="Height" bind:value={height} placeholder={inherited?.desktop_height ? `Inherited: ${inherited.desktop_height}` : '1080'} inherited={inherited?.desktop_height || ''} />
  <LabeledSelect id="desktop_color_depth" label="Color Depth" bind:value={colorDepth} options={depthOptions} inherited={inherited?.desktop_color_depth || ''} />
</div>
