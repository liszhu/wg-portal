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
        UserIdentifier: (state) => state.user || 'unknown',
        LoginProviders: (state) => state.providers,
        IsAuthenticated: (state) => state.user != null,
        ReturnUrl: (state) => state.returnUrl || '/',
    },
    actions: {
        SetReturnUrl(link) {
            this.returnUrl = link
            localStorage.setItem('returnUrl', link)
        },
        ResetReturnUrl() {
            this.returnUrl = null
            localStorage.removeItem('returnUrl')
        },
        // LoadProviders always returns a fulfilled promise, even if the request failed.
        async LoadProviders() {
            fetchWrapper.get(`${baseUrl}/providers`)
                .then(providers => this.providers = providers)
                .catch(error => {
                    this.providers = []
                    console.log("Failed to load auth providers: ", error)
                    notify({
                        title: "Backend Connection Failure",
                        text: "Failed to load external authentication providers!",
                    })
                })
        },

        // LoadSession returns promise that might have been rejected if the session was not authenticated.
        async LoadSession() {
            return fetchWrapper.get(`${baseUrl}/session`)
                .then(session => {
                    if (session.LoggedIn === true) {
                        this.ResetReturnUrl()
                        this.setUserInfo(session.UserIdentifier)
                        return session.UserIdentifier
                    } else {
                        this.setUserInfo(null)
                        return Promise.reject(new Error('session not authenticated'))
                    }
                })
                .catch(err => {
                    this.setUserInfo(null)
                    return Promise.reject(err)
                })
        },
        // Login returns promise that might have been rejected if the login attempt was not successful.
        async Login(username, password) {
            return fetchWrapper.post(`${baseUrl}/login`, { username, password })
                .then(user =>  {
                    this.ResetReturnUrl()
                    this.setUserInfo(user.Identifier)
                    return user.Identifier
                })
                .catch(err => {
                    console.log("Login failed:", err)
                    this.setUserInfo(null)
                    return Promise.reject(new Error("login failed"))
                })
        },
        async Logout() {
            this.setUserInfo(null)
            this.ResetReturnUrl() // just to be sure^^

            try {
                await fetchWrapper.get(`${baseUrl}/logout`)
            } catch (e) {
                console.log("Logout request failed:", e)
            }

            notify({
                title: "Logged Out",
                text: "Logout successful!",
                type: "warn",
            })


            await router.push('/login')
        },
        // -- internal setters
        setUserInfo(uid) {
            this.user = uid
            // store user details and jwt in local storage to keep user logged in between page refreshes
            if (uid) {
                localStorage.setItem('user', JSON.stringify(uid))
            } else {
                localStorage.removeItem('user')
            }
        },
    }
});