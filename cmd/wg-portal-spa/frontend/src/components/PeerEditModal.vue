<script setup>
import Modal from "./Modal.vue";
import {peerStore} from "../stores/peers";
import {interfaceStore} from "../stores/interfaces";
import {computed} from "vue";
import { useI18n } from 'vue-i18n';

const { t } = useI18n()

const peers = peerStore()
const interfaces = interfaceStore()

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

const selectedInterface = computed(() => interfaces.GetSelected)

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }
  if (selectedInterface.value.Mode === "server") {
    if (selectedPeer.value) {
      return t("interfaces.peer.edit") + ": " + selectedPeer.value.Name
    }
    return t("interfaces.peer.new")
  } else {
    if (selectedPeer.value) {
      return t("interfaces.endpoint.edit") + ": " + selectedPeer.value.Name
    }
    return t("interfaces.endpoint.new")
  }
})

</script>

<template>
  <Modal :title="title" :visible="visible" @close="close">
    <template #default>
      <fieldset>
        <legend class="mt-4">General</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.displayname') }}</label>
          <input type="text" class="form-control" placeholder="Displayname of the peer">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.linkeduser') }}</label>
          <input type="text" class="form-control" placeholder="Linked user">
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">Cryptography</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.privatekey') }}</label>
          <input type="email" class="form-control" placeholder="The private key" required>
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.publickey') }}</label>
          <input type="email" class="form-control" placeholder="The public key" required>
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.presharedkey') }}</label>
          <input type="email" class="form-control" placeholder="Optional pre-shared key">
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">Networking</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.ips') }}</label>
          <input type="text" class="form-control" placeholder="Client IP Address">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.allowedips') }}</label>
          <input type="text" class="form-control" placeholder="Allowed IP Address">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.extraallowedips') }}</label>
          <input type="text" class="form-control" placeholder="Extra Allowed IP's (Server Sided)">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.peeredit.dns') }}</label>
          <input type="text" class="form-control" placeholder="Client DNS Servers">
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.peeredit.persistendkeepalive') }}</label>
            <input type="number" class="form-control" placeholder="Persistent Keepalive (0 = off)">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.peeredit.mtu') }}</label>
            <input type="number" class="form-control" placeholder="Client MTU (0 = default)">
          </div>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">State</legend>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox">
          <label class="form-check-label" >Disabled</label>
        </div>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox" checked="">
          <label class="form-check-label">Ignore global settings</label>
        </div>
      </fieldset>
    </template>
    <template #footer>
      <div class="flex-fill text-start">
        <button type="button" class="btn btn-danger me-1">Delete</button>
      </div>
      <button type="button" class="btn btn-primary me-1">Save</button>
      <button @click.prevent="close" type="button" class="btn btn-secondary">Discard</button>
    </template>
  </Modal>
</template>

<style>
.config-qr-img {
  max-width: 100%;
}
</style>
