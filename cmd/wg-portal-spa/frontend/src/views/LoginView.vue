<script setup>

import {computed, ref} from "vue";
import {authStore} from "../stores/auth";
import router from '../router/index.js'
import {notify} from "@kyvg/vue3-notification";

const auth = authStore()

const loggingIn = ref(false)
const username = ref("")
const password = ref("")

const usernameInvalid = computed(() => username.value === "")
const passwordInvalid = computed(() => password.value === "")
const disableLoginBtn = computed(() => username.value === "" || password.value === "" || loggingIn.value)

const login = function () {
  console.log("Performing login for user:", username.value)
  loggingIn.value = true
  auth.Login(username.value, password.value)
      .then(uid => {
        notify({
          title: "Logged in",
          text: "Authentication succeeded!",
          type: 'success',
        })
        loggingIn.value = false
        router.push(auth.ReturnUrl)
      })
      .catch(error => {
        notify({
          title: "Login failed!",
          text: "Authentication failed!",
          type: 'error',
        })

        // delay the user from logging in for a short amount of time
        setTimeout(() => loggingIn.value = false, 1000);
      })
}

const externalLogin = function (provider) {
  console.log("Performing external login for provider", provider.Identifier)
  loggingIn.value = true
  console.log(router.currentRoute.value)
  let currentUri = window.location.origin + "/#" + router.currentRoute.value.fullPath
  let redirectUrl = `${WGPORTAL_BACKEND_BASE_URL}${provider.ProviderUrl}`
  redirectUrl += "?redirect=true"
  redirectUrl += "&return=" + encodeURIComponent(currentUri)
  window.location.href = redirectUrl;
}
</script>

<template>
  <div class="row">
    <div class="col-lg-3"></div><!-- left spacer -->
    <div class="col-lg-6">
      <div class="card mt-5">
        <div class="card-header">Please sign in <div class="float-end">
          <RouterLink class="nav-link" :to="{ name: 'home' }" title="Home"><i class="fas fa-times-circle"></i></RouterLink>
        </div></div>
        <div class="card-body">
          <form method="post">
            <fieldset>
              <div class="form-group">
                <label for="inputUsername" class="form-label">{{ $t('login.username') }}</label>
                <div class="input-group mb-3">
                  <span class="input-group-text"><span class="far fa-user p-2"></span></span>
                  <input type="text" name="username" class="form-control" id="inputUsername" aria-describedby="usernameHelp"
                         :class="{'is-invalid':usernameInvalid, 'is-valid':!usernameInvalid}"
                         :placeholder="$t('login.userMessage')" v-model="username">
                </div>
              </div>
              <div class="form-group">
                <label for="inputPassword" class="form-label">{{ $t('login.pass') }}</label>
                <div class="input-group mb-3">
                  <span class="input-group-text"><span class="fas fa-lock p-2"></span></span>
                  <input type="password" name="password" class="form-control" id="inputPassword" :placeholder="$t('login.passMessage')"
                         :class="{'is-invalid':passwordInvalid, 'is-valid':!passwordInvalid}" v-model="password">
                </div>
              </div>

              <div class="row mt-5 d-flex">
                <div class="d-flex mb-2" :class="{'col-lg-4':auth.LoginProviders.length < 3, 'col-lg-12':auth.LoginProviders.length >= 3}">
                  <button class="btn btn-primary flex-fill" type="submit" :disabled="disableLoginBtn" @click.prevent="login">
                    {{ $t('login.btn') }} <i v-if="loggingIn" class="ms-2 fa-solid fa-circle-notch fa-spin"></i>
                  </button>
                </div>
                <div class="d-flex mb-2" :class="{'col-lg-8':auth.LoginProviders.length < 3, 'col-lg-12':auth.LoginProviders.length >= 3}">
                  <!-- OpenIdConnect / OAUTH providers -->
                    <button v-for="(provider, idx) in auth.LoginProviders" :key="provider.Identifier" :disabled="loggingIn"
                       class="btn btn-outline-primary flex-fill" :class="{'ms-1':idx > 0}" :title="provider.Name"
                            @click.prevent="externalLogin(provider)" v-html="provider.Name"></button>
                </div>
              </div>

              <div class="mt-3">
              </div>
            </fieldset>
          </form>


        </div>
      </div>
    </div>
    <div class="col-lg-3"></div><!-- right spacer -->
  </div>
</template>
