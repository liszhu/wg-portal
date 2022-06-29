import { defineStore } from 'pinia'

import { notify } from "@kyvg/vue3-notification";
import { fetchWrapper } from '../helpers/fetch-wrapper.js'
import router from '../router/index.js'

const baseUrl = `${WGPORTAL_BACKEND_BASE_URL}/auth`

export const authStore = defineStore({
    id: 'auth',
    state: () => ({
        // initialize state from local storage to enable user to stay logged in
        user: JSON.parse(localStorage.getItem('user')),
        providers: [],
        returnUrl: localStorage.getItem('returnUrl')
    }),
    getters: {
        LoginProviders: (state) => state.providers,
        All: (state) => state.interfaces,
        GetSelected: (state) => state.interfaces.find((i) => i.Identifier === state.selected),
        isFetching: (state) => state.fetching,
    },
    actions: {
        setReturnUrl(link) {
            if (!localStorage.getItem('returnUrl')) {
                localStorage.setItem('returnUrl', link)
            }
        },
        resetReturnUrl() {
            localStorage.setItem('returnUrl', '')
        },
        setUserInfo(uid) {
            this.user = uid
            // store user details and jwt in local storage to keep user logged in between page refreshes
            if (uid) {
                localStorage.setItem('user', JSON.stringify(uid))
            } else {
                localStorage.removeItem('user')
            }
        },
        async loadProviders() {
            fetchWrapper.get(`${baseUrl}/providers`)
                .then(providers => this.providers = providers)
                .catch(error => {
                    console.log("Failed to load auth providers: ", error)
                    notify({
                        title: "Backend Connection Failure",
                        text: "Failed to load external authentication providers!",
                    })
                })
        },
        async loginOauth() {
            const session = await fetchWrapper.get(`${baseUrl}/session`)

            if (session.LoggedIn) {
                this.setUserInfo(user.Identifier)

                notify({
                    title: "Logged in",
                    text: "Authentication suceeded!",
                    type: 'success',
                })
            } else {
                this.setUserInfo(null)
            }

            // redirect to previous url or default to home page
            let returnUrl = this.returnUrl
            this.resetReturnUrl()
            return returnUrl || '/'
        },
        async login(username, password) {
            const user = await fetchWrapper.post(`${baseUrl}/login`, { username, password })

            this.setUserInfo(user.Identifier)

            // redirect to previous url or default to home page
            let returnUrl = this.returnUrl
            this.resetReturnUrl()
            await router.push(returnUrl || '/')
        },
        async logout() {
            await fetchWrapper.get(`${baseUrl}/logout`)

            this.setUserInfo(null)

            notify({
                title: "Logged Out",
                text: "Logout successful!",
                type: "warn",
            })

            this.resetReturnUrl()
            await router.push('/login')
        }
    }
});