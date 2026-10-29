<script lang="ts">
import type { Fill } from './fill.svelte';
import { entryText } from './labels';

let { fill }: { fill: Fill } = $props();

const id = $props.id();
</script>

<!--
  Opened as a modal, so the page behind it can't be reached and Escape closes it; as it closes, the browser gives the
  focus back to the form's Log button, or the attachment does where the browser doesn't.
-->
<dialog
  class="dialog"
  role="alertdialog"
  aria-labelledby="{id}-title"
  aria-describedby="{id}-text"
  {@attach fill.dialog}
>
  <h2 class="dialog-title" id="{id}-title">Log these hours?</h2>
  <p class="dialog-text" id="{id}-text">{fill.entry && entryText(fill.entry)}</p>
  <!-- The focus starts on the choice that changes nothing: a log writes to Jira, and only Jira can take it back. -->
  <button class="form-button" type="button" autofocus onclick={() => fill.cancel()}>Cancel</button>
  <button class="form-button form-button-primary" type="button" onclick={() => fill.log()}>Log</button>
</dialog>
