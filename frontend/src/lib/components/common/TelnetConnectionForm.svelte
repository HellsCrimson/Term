<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';

  interface Props {
    host: string;
    port: string;
    username: string;
    password: string;
  }

  let {
    host = $bindable(''),
    port = $bindable('23'),
    username = $bindable(''),
    password = $bindable(''),
  }: Props = $props();
</script>

<div class="grid grid-cols-2 gap-3">
  <div class="col-span-2">
    <LabeledInput id="telnet_host" label="Host *" bind:value={host} placeholder="192.168.1.100 or telnet-server.local" />
  </div>
  <LabeledInput id="telnet_port" label="Port" bind:value={port} placeholder="23" />
  <LabeledInput id="telnet_username" label="Username (Optional)" bind:value={username} placeholder="admin" />
  <div class="col-span-2">
    <LabeledInput id="telnet_password" label="Password (Optional)" type="password" bind:value={password} placeholder="••••••••" />
  </div>
</div>

