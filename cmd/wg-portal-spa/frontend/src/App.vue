<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<script>
export default {
  methods: {
    switchLanguage(lang) {
      if (this.$i18n.locale !== lang) {
        localStorage.setItem('wgLang', lang)
        this.$i18n.locale = lang
      }
    }
  },
  computed: {
    languageFlag() {
      // `this` points to the component instance
      let lang = this.$i18n.locale.toLowerCase()
      if (lang === "en") {
        lang = "us"
      }
      return "fi-" + lang
    }
  }
}
</script>

<template>
  <notifications position="top right" :duration="3000" :ignore-duplicates="true" />

  <nav class="navbar navbar-expand-lg navbar-dark bg-primary">
    <div class="container-fluid">
      <a class="navbar-brand" href="/"><img src="/img/header-logo.png" alt="WireGuard Portal" /></a>
      <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarTop"
        aria-controls="navbarColor01" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>

      <div class="collapse navbar-collapse" id="navbarTop">
        <ul class="navbar-nav me-auto">
          <li class="nav-item">
            <RouterLink class="nav-link" :to="{ name: 'home' }">{{ $t('menu.home') }}</RouterLink>
          </li>
          <li class="nav-item">
            <RouterLink class="nav-link" :to="{ name: 'interfaces' }">{{ $t('menu.interfaces') }}</RouterLink>
          </li>
          <li class="nav-item">
            <RouterLink class="nav-link" :to="{ name: 'users' }">{{ $t('menu.users') }}</RouterLink>
          </li>
        </ul>

        <div class="navbar-nav d-flex justify-content-end">
          <div class="nav-item dropdown">
            <a class="nav-link dropdown-toggle" data-bs-toggle="dropdown" href="#" role="button" aria-haspopup="true"
              aria-expanded="false">{{ $t('menu.firstname') }} {{ $t('menu.lastname') }}</a>
            <!-- <li class="nav-item">
            <RouterLink class="nav-link" :to="{ name: 'users' }">{{ $t('hello') }}</RouterLink>
          </li> -->
            <div class="dropdown-menu">
              <a class="dropdown-item" href="/admin"><i class="fas fa-cogs"></i> {{ $t('menu.admin') }}</a>
              <a class="dropdown-item" href="/admin/users"><i class="fas fa-users-cog"></i> {{ $t('menu.user') }}</a>
              <div class="dropdown-divider"></div>
              <a class="dropdown-item" href="/user/profile"><i class="fas fa-user"></i> {{ $t('menu.profile') }}</a>
              <div class="dropdown-divider"></div>
              <a class="dropdown-item" href="/auth/logout"><i class="fas fa-sign-out-alt"></i> {{ $t('menu.logout')
              }}</a>
            </div>
          </div>
          <div class="nav-item">
            <a href="/login" class="nav-link"><i class="fas fa-sign-in-alt fa-sm fa-fw me-2"></i>{{ $t('menu.login')
            }}</a>
          </div>
        </div>
      </div>
    </div>
  </nav>

  <div class="container mt-5 flex-shrink-0">
    <RouterView />
  </div>

  <footer class="page-footer mt-auto">
    <div class="container mt-5">
      <div class="row align-items-center">
        <div class="col-6">Powered by <img height="20" src="@/assets/logo.svg" alt="Vue.JS" /></div>
        <div class="col-6 text-end">
          <div class="btn-group" role="group" aria-label="{{ $t('menu.lang') }}">
            <div class="btn-group" role="group">
              <button type="button" class="btn btn btn-secondary pe-0" data-bs-toggle="dropdown" aria-haspopup="true" aria-expanded="false"><span class="fi" :class="languageFlag"></span></button>
              <div class="dropdown-menu" aria-labelledby="btnGroupDrop3" style="">
                <a class="dropdown-item" @click.prevent="switchLanguage('en')" href="#"><span class="fi fi-us"></span> English</a>
                <a class="dropdown-item" @click.prevent="switchLanguage('de')" href="#"><span class="fi fi-de"></span> Deutsch</a>
                <a class="dropdown-item" @click.prevent="switchLanguage('es')" href="#"><span class="fi fi-es"></span> Español</a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </footer>
</template>

<style>
.vue-notification-group {
  margin-top:5px;
}
</style>