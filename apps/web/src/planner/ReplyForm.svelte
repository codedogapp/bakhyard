<script lang="ts">
import { sendLabel } from './labels';
import { ReplyState } from './reply.svelte';

// The plan, by its ID as its page's path holds it, that the reply goes to.
let { id }: { id: string } = $props();

const reply = new ReplyState();

function submit(event: SubmitEvent): void {
  event.preventDefault();
  void reply.send(id);
}
</script>

<!-- No maxlength: the service limits the message (decision 414), and a refused one stays in the field to edit. -->
<form class="form-row" onsubmit={submit}>
  <textarea
    class="form-field"
    aria-label="Reply"
    placeholder="Reply to Copilot"
    rows="3"
    readonly={reply.busy}
    bind:value={reply.text}
  ></textarea>
  <!-- aria-disabled, not disabled, so focus stays on the button while it waits. -->
  <button class="form-button" type="submit" aria-disabled={reply.busy}>Send</button>
  <p class="form-status" role="status">{sendLabel(reply.status)}</p>
</form>

<style>
/* Under the conversation, the form leaves the space that a section's head leaves over it. */
form {
  margin-top: 30px;
}

/* The reply takes the row, with Send under it. */
textarea {
  flex-basis: 100%;
  margin-bottom: 12px;
  resize: vertical;
}
</style>
