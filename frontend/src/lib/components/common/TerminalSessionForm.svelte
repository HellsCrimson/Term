<script lang="ts">
  import LabeledInput from './LabeledInput.svelte';
  import LabeledTextarea from './LabeledTextarea.svelte';

  interface InheritedCfg {
    working_directory?: string;
    startup_commands?: string;
    environment_variables?: string;
  }

  interface Props {
    workingDirectory: string;
    startupCommands: string;
    environmentVariables: string;
    inherited?: InheritedCfg;
    dirPlaceholder?: string;
    commandsPlaceholder?: string;
    envPlaceholder?: string;
    rowsCommands?: number;
    rowsEnv?: number;
  }

  let {
    workingDirectory = $bindable(''),
    startupCommands = $bindable(''),
    environmentVariables = $bindable(''),
    inherited,
    dirPlaceholder = '~/projects or /home/user',
    commandsPlaceholder = 'cd ~/project; source .env',
    envPlaceholder = 'KEY1=value1; KEY2=value2',
    rowsCommands = 2,
    rowsEnv = 2,
  }: Props = $props();
</script>

<div class="space-y-3">
  <LabeledInput
    id="working_directory"
    label="Working Directory"
    bind:value={workingDirectory}
    placeholder={inherited?.working_directory ? `Inherited: ${inherited.working_directory}` : dirPlaceholder}
    inherited={inherited?.working_directory || ''}
  />
  <LabeledTextarea
    id="startup_commands"
    label="Startup Commands"
    bind:value={startupCommands}
    rows={rowsCommands}
    placeholder={inherited?.startup_commands ? `Inherited: ${inherited.startup_commands}` : commandsPlaceholder}
    inherited={inherited?.startup_commands || ''}
    hint={!inherited?.startup_commands ? 'Commands to run when the session starts (separated by semicolons)' : ''}
  />
  <LabeledTextarea
    id="environment_variables"
    label="Environment Variables"
    bind:value={environmentVariables}
    rows={rowsEnv}
    placeholder={inherited?.environment_variables ? `Inherited: ${inherited.environment_variables}` : envPlaceholder}
    inherited={inherited?.environment_variables || ''}
    hint={!inherited?.environment_variables ? 'Environment variables (KEY=value; separated by semicolons)' : ''}
  />
</div>

