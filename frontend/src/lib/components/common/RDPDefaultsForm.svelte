<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';
  import LabeledSelect from './LabeledSelect.svelte';
  import DisplaySettings from './DisplaySettings.svelte';

  type Security = 'any' | 'nla' | 'tls' | 'rdp';

  interface InheritedCfg {
    rdp_username?: string;
    rdp_domain?: string;
    rdp_security?: string;
    desktop_width?: string;
    desktop_height?: string;
    desktop_color_depth?: string;
  }

  interface Props {
    username: string;
    domain: string;
    security: Security;
    width: string;
    height: string;
    colorDepth: string;
    inherited?: InheritedCfg;
  }

  let { username, domain, security, width, height, colorDepth, inherited }: Props = $props();

  const securityOptions = [
    { value: 'any', label: 'Any' },
    { value: 'nla', label: 'NLA' },
    { value: 'tls', label: 'TLS' },
    { value: 'rdp', label: 'RDP' }
  ];
</script>

<div class="space-y-3">
  <div class="grid grid-cols-2 gap-3">
    <LabeledInput id="rdp_username" label="RDP Username" bind:value={username} placeholder={inherited?.rdp_username ? `Inherited: ${inherited.rdp_username}` : 'administrator'} inherited={inherited?.rdp_username || ''} />
    <LabeledInput id="rdp_domain" label="RDP Domain" bind:value={domain} placeholder={inherited?.rdp_domain ? `Inherited: ${inherited.rdp_domain}` : 'CORP'} inherited={inherited?.rdp_domain || ''} />
  </div>
  <LabeledSelect id="rdp_security" label="RDP Security" bind:value={security} options={securityOptions} inherited={inherited?.rdp_security || ''} />
  <div class="pt-3 border-t border-gray-600">
    <h5 class="text-xs font-medium text-gray-300 mb-2">Display Settings</h5>
    <DisplaySettings bind:width={width} bind:height={height} bind:colorDepth={colorDepth} inherited={inherited} />
  </div>
</div>
