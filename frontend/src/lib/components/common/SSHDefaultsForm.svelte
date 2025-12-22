<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';

  interface InheritedCfg {
    ssh_username?: string;
    ssh_port?: string;
    ssh_key_path?: string;
  }

  interface Props {
    username: string;
    port: string;
    keyPath: string;
    inherited?: InheritedCfg;
  }

  let {
    username = $bindable(''),
    port = $bindable('22'),
    keyPath = $bindable(''),
    inherited
  }: Props = $props();
</script>

<div class="space-y-3">
  <LabeledInput
    id="ssh_username"
    label="SSH Username"
    bind:value={username}
    placeholder={inherited?.ssh_username ? `Inherited: ${inherited.ssh_username}` : 'root (inherited by SSH sessions)'}
    inherited={inherited?.ssh_username || ''}
  />
  <LabeledInput
    id="ssh_port"
    label="SSH Port"
    bind:value={port}
    placeholder={inherited?.ssh_port ? `Inherited: ${inherited.ssh_port}` : '22 (inherited by SSH sessions)'}
    inherited={inherited?.ssh_port || ''}
  />
  <LabeledInput
    id="ssh_key_path"
    label="SSH Key Path"
    bind:value={keyPath}
    placeholder={inherited?.ssh_key_path ? `Inherited: ${inherited.ssh_key_path}` : '~/.ssh/id_rsa (inherited by SSH sessions)'}
    inherited={inherited?.ssh_key_path || ''}
    hint={!inherited?.ssh_key_path ? 'Path to your private key file' : ''}
  />
</div>

