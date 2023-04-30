export class LoginService {
    private token = ""

    public setToken(token: string) {
        this.token = token
    }

    public getToken(): string {
        return this.token
    }

    public loggedIn(): boolean {
        return this.token != ""
    }
}