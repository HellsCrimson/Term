<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';

  interface Props {
    host: string;
    port: string;
    password: string;
  }

  let {
    host = $bindable(''),
    port = $bindable('5900'),
    password = $bindable(''),
  }: Props = $props();
</script>

<div class="grid grid-cols-2 gap-3">
  <div class="col-span-2">
    <LabeledInput id="vnc_host" label="Host *" bind:value={host} placeholder="192.168.1.100 or vnc-server.local" />
  </div>
  <LabeledInput id="vnc_port" label="Port" bind:value={port} placeholder="5900" />
  <LabeledInput id="vnc_password" label="Password" type="password" bind:value={password} placeholder="••••••••" />
</div>

