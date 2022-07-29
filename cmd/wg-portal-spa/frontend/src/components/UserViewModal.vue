<script setup>
import Modal from "./Modal.vue";
import {userStore} from "../stores/users";
import {computed, ref, watch} from "vue";
import { useI18n } from 'vue-i18n';
import { notify } from "@kyvg/vue3-notification";

const { t } = useI18n()

const users = userStore()

const props = defineProps({
  userId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

const selectedUser = computed(() => {
  return users.Find(props.userId)
})

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }
  return t("users.view") + ": " + selectedUser.value.Identifier
})

const userPeers = computed(() => {
  return []
})

const formData = ref(freshFormData())

function freshFormData() {
  return {
    Disabled: false,
    IgnoreGlobalSettings: true,

    Endpoint: {
      Value: "",
      Overridable: false,
    },
    AllowedIPsStr: {
      Value: "",
      Overridable: false,
    },
    ExtraAllowedIPsStr: "",
    PrivateKey: "",
    PublicKey: "",
    PresharedKey: "",
    PersistentKeepalive: {
      Value: 0,
      Overridable: false,
    },

    DisplayName: "",
    Identifier: "",
    UserIdentifier: "",

    InterfaceConfig: {
      PublicKey: {
        Value: "",
        Overridable: false,
      },
      AddressStr: {
        Value: "",
        Overridable: false,
      },
      DnsStr: {
        Value: "",
        Overridable: false,
      },
      DnsSearchStr: {
        Value: "",
        Overridable: false,
      },
      Mtu: {
        Value: 0,
        Overridable: false,
      },
      FirewallMark: {
        Value: 0,
        Overridable: false,
      },
      RoutingTable: {
        Value: "",
        Overridable: false,
      },
      PreUp: {
        Value: "",
        Overridable: false,
      },
      PostUp: {
        Value: "",
        Overridable: false,
      },
      PreDown: {
        Value: "",
        Overridable: false,
      },
      PostDown: {
        Value: "",
        Overridable: false,
      },
    }
  }
}

// functions

watch(() => props.visible, async (newValue, oldValue) => {
      if (oldValue === false && newValue === true) { // if modal is shown
        if (!selectedUser.value) {
          await loadNewUserData()
        }
      }
    }
)

async function loadNewUserData() {
  console.log("loading new user data...")
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
        <legend>Peer List</legend>
        <div v-if="userPeers.length===0">
          <p>{{ $t('users.nopeers.message') }}</p>
        </div>
        <table v-if="userPeers.length!==0" class="table table-sm" id="peerTable">
          <thead>
          <tr>
            <th scope="col">{{ $t('interfaces.tableHeadings[0]') }}</th>
            <th scope="col">{{ $t('interfaces.tableHeadings[1]') }}</th>
            <th scope="col">{{ $t('interfaces.tableHeadings[2]') }}</th>
            <th scope="col">{{ $t('interfaces.tableHeadings[3]') }}</th>
            <th scope="col" v-if="interfaces.GetSelected.Mode==='client'">{{ $t('interfaces.tableHeadings[4]') }}</th>
            <th scope="col">{{ $t('interfaces.tableHeadings[5]') }}</th>
            <th scope="col"></th><!-- Actions -->
          </tr>
          </thead>
          <tbody>
          <tr v-for="peer in userPeers" :key="peer.Identifier">
            <td>{{peer.Name}}</td>
            <td>{{peer.Identifier}}</td>
            <td>{{peer.User}}</td>
            <td>
              <span v-for="ip in peer.IPs" :key="ip" class="badge rounded-pill bg-light">{{ ip }}</span>
            </td>
            <td v-if="interfaces.GetSelected.Mode==='client'">{{peer.Endpoint}}</td>
            <td>{{peer.LastConnected}}</td>
            <td class="text-center">
              <a @click.prevent="viewedPeerId=peer.Identifier" href="#" title="Download config"><i class="fas fa-eye me-2"></i></a>
              <a @click.prevent="editPeerId=peer.Identifier" href="#" title="Email config"><i class="fas fa-cog"></i></a>
            </td>
          </tr>
          </tbody>
        </table>

      </fieldset>
    </template>
    <template #footer>
      <button @click.prevent="close" type="button" class="btn btn-primary">Close</button>
    </template>
  </Modal>
</template>
