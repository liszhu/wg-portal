<script setup>
import Modal from "./Modal.vue";
import {userStore} from "../stores/users";
import {interfaceStore} from "../stores/interfaces";
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
  if (selectedUser.value) {
    return t("users.edit") + ": " + selectedUser.value.Identifier
  }
  return t("users.new")
})

const formData = ref(freshFormData())

function freshFormData() {
  return {
    Identifier: "",

    Email: "",
    Source: "",
    IsAdmin: false,

    Firstname: "",
    Lastname: "",
    Phone: "",
    Department: "",
    Notes: "",

    Password: "",

    Disabled: false,
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
        <legend class="mt-4">General</legend>
        <div class="form-group" v-if="props.userId==='#NEW#'">
          <label class="form-label mt-4">{{ $t('modals.useredit.identifier') }}</label>
          <input type="text" class="form-control" placeholder="The user id" v-model="formData.Identifier">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.useredit.source') }}</label>
          <input type="text" class="form-control" placeholder="The user source" v-model="formData.Source" disabled="disabled">
        </div>
        <div class="form-group" v-if="formData.Source==='db'">
          <label class="form-label mt-4">{{ $t('modals.useredit.password') }}</label>
          <input type="text" class="form-control" placeholder="Password" aria-describedby="passwordHelp" v-model="formData.Password">
          <small id="passwordHelp" class="form-text text-muted">Leave this field blank to keep current password.</small>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">User Information</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.useredit.email') }}</label>
          <input type="email" class="form-control" placeholder="Email" v-model="formData.Email">
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.useredit.firstname') }}</label>
            <input type="text" class="form-control" placeholder="Firstname" v-model="formData.Firstname">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.useredit.lastname') }}</label>
            <input type="text" class="form-control" placeholder="Lastname" v-model="formData.Lastname">
          </div>
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.useredit.phone') }}</label>
            <input type="text" class="form-control" placeholder="Phone" v-model="formData.Phone">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.useredit.department') }}</label>
            <input type="text" class="form-control" placeholder="Department" v-model="formData.Department">
          </div>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">Notes</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.useredit.notes') }}</label>
          <textarea class="form-control" rows="2" v-model="formData.Notes"></textarea>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">State</legend>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox" v-model="formData.Disabled">
          <label class="form-check-label" >Disabled</label>
        </div>
        <div class="form-check form-switch">
          <input class="form-check-input" type="checkbox" checked="" v-model="formData.IsAdmin">
          <label class="form-check-label">Is Admin</label>
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
