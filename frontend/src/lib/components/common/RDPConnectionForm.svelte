<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';
  import LabeledSelect from './LabeledSelect.svelte';

  type Security = 'any' | 'nla' | 'tls' | 'rdp';

  interface Props {
    host: string;
    port: string;
    security: Security;
    username: string;
    password: string;
    domain: string;
  }

  let {
    host = $bindable(''),
    port = $bindable('3389'),
    security = $bindable('any'),
    username = $bindable(''),
    password = $bindable(''),
    domain = $bindable(''),
  }: Props = $props();

  const securityOptions = [
    { value: 'any', label: 'Any' },
    { value: 'nla', label: 'NLA' },
    { value: 'tls', label: 'TLS' },
    { value: 'rdp', label: 'RDP' }
  ];
</script>

<div class="grid grid-cols-2 gap-3">
  <div class="col-span-2">
    <LabeledInput id="rdp_host" label="Host *" bind:value={host} placeholder="192.168.1.100 or windows-server.local" />
  </div>
  <LabeledInput id="rdp_port" label="Port" bind:value={port} placeholder="3389" />
  <LabeledSelect id="rdp_security" label="Security" bind:value={security} options={securityOptions} />
  <LabeledInput id="rdp_username" label="Username" bind:value={username} placeholder="administrator" />
  <LabeledInput id="rdp_password" label="Password" type="password" bind:value={password} placeholder="••••••••" />
  <div class="col-span-2">
    <LabeledInput id="rdp_domain" label="Domain (Optional)" bind:value={domain} placeholder="CORP or corp.local" />
  </div>
  
</div>

