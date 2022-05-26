<script setup>
import Modal from "./Modal.vue";
import {peerStore} from "../stores/peers";
import {storeToRefs} from "pinia";
import {computed} from "vue";

const peers = peerStore()

const props = defineProps({
  peerId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

function close() {
  emit('close')
}

const selectedPeer = computed(() => {
  return peers.Find(props.peerId)
})

const title = computed(() => {
  if (selectedPeer.value) {
    return "Peer: " + selectedPeer.value.Name
  }
  return ""
})

</script>

<template>
  <Modal :title="title" :visible="visible" @close="close">
  </Modal>
</template>

