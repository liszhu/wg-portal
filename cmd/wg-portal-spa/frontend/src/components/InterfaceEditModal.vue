<script setup>
import Modal from "./Modal.vue";
import {interfaceStore} from "../stores/interfaces";
import {computed, ref, watch} from "vue";
import { useI18n } from 'vue-i18n';
import { notify } from "@kyvg/vue3-notification";

const { t } = useI18n()

const interfaces = interfaceStore()

const props = defineProps({
  interfaceId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

const selectedInterface = computed(() => {
  return interfaces.Find(props.interfaceId)
})

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }

  if (selectedInterface.value) {
    return t("interfaces.interface.edit") + ": " + selectedInterface.value.Identifier
  }
  return t("interfaces.interface.new")
})

const formData = ref(freshFormData())

function freshFormData() {
  return {
    Disabled: false,
    DisplayName: "",
    Identifier: "",
    Type: "server",

    PublicKey: "",
    PrivateKey: "",

    ListenPort:  51820,
    AddressStr: "",
    DnsStr: "",
    DnsSearchStr: "",

    Mtu: 0,
    FirewallMark: 0,
    RoutingTable: "",

    PreUp: "",
    PostUp: "",
    PreDown: "",
    PostDown: "",

    SaveConfig: false

    // TODO: peer default stuff
  }
}

// functions

watch(() => props.visible, async (newValue, oldValue) => {
      if (oldValue === false && newValue === true) { // if modal is shown
        console.log(selectedInterface.value)
        if (!selectedInterface.value) {
          await interfaces.PrepareInterface()

          // fill form data
          formData.value.Identifier = interfaces.Prepared.Identifier
          formData.value.PublicKey = interfaces.Prepared.PublicKey
          formData.value.PrivateKey = interfaces.Prepared.PrivateKey
        }
      }
    }
)

async function loadNewInterfaceData() {
  console.log("loading new interface data...")
  notify({
    title: "Authorization",
    text: "You have been logged in!",
  })
  notify({
    title: "Authorization2",
    text: "You have been logged in!",
  })
  notify({
    title: "Authorization3",
    text: "You have been logged in!",
  })

}

function close() {
  formData.value = freshFormData()
  emit('close')
}

</script>

<template>
  <Modal :title="title" :visible="visible" @close="close">
    <template #default>
      <fieldset>
        <legend class="mt-4">General</legend>
        <div class="form-group" v-if="props.interfaceId==='#NEW#'">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.identifier') }}</label>
          <input type="text" class="form-control" placeholder="The device identifier" v-model="formData.Identifier">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.displayname') }}</label>
          <select class="form-select" v-model="formData.Type">
            <option value="server">Server Mode</option>
            <option value="client">Client Mode</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.displayname') }}</label>
          <input type="text" class="form-control" placeholder="A descriptive name of the interface" v-model="formData.DisplayName">
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">Cryptography</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.privatekey') }}</label>
          <input type="email" class="form-control" placeholder="The private key" required v-model="formData.PrivateKey">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.publickey') }}</label>
          <input type="email" class="form-control" placeholder="The public key" required v-model="formData.PublicKey">
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">Networking</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.ips') }}</label>
          <input type="text" class="form-control" placeholder="IP Address" v-model="formData.AddressStr">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.listenport') }}</label>
          <input type="text" class="form-control" placeholder="Listen Port" v-model="formData.ListenPort">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.dns') }}</label>
          <input type="text" class="form-control" placeholder="DNS Servers" v-model="formData.DnsStr">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.interfaceedit.dnssearch') }}</label>
          <input type="text" class="form-control" placeholder="DNS Search prefix" v-model="formData.DnsSearchStr">
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.interfaceedit.mtu') }}</label>
            <input type="number" class="form-control" placeholder="Client MTU (0 = default)" v-model="formData.Mtu">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.interfaceedit.firewallmark') }}</label>
            <input type="number" class="form-control" placeholder="Firewall Mark (0 = default)" v-model="formData.FirewallMark">
          </div>
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.interfaceedit.routingtable') }}</label>
            <input type="number" class="form-control" placeholder="Routing Table (0 = default)" v-model="formData.RoutingTable">
          </div>
          <div class="form-group col-md-6">
          </div>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">State</legend>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox" v-model="formData.Disabled">
          <label class="form-check-label" >Disabled</label>
        </div>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox" checked="" v-model="formData.SaveConfig">
          <label class="form-check-label">Save Config to File</label>
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
