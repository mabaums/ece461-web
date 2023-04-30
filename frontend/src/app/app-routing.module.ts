import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AppComponent } from './app.component';
import { PackageResultsComponent } from './package-results/package-results.component';
import { AdvancedSearchComponent } from './advanced-search/advanced-search.component';
import { HomePageComponent } from './home-page/home-page.component';
import { PackagePageComponent } from './package-page/package-page.component';
import { CreatePageComponent } from './create-page/create-page.component';
import { UpdatePageComponent } from './update-page/update-page.component';
import { LoginPageComponent } from './login-page/login-page.component';

const routes: Routes = [
  { path: '', component: HomePageComponent},
  { path: 'search', component: PackageResultsComponent},
  { path: 'search/advanced', component: AdvancedSearchComponent},
  { path: 'package', component: PackagePageComponent},
  { path: 'create', component: CreatePageComponent},
  { path: 'update', component: UpdatePageComponent}, 
  { path: 'login', component: LoginPageComponent},
  { path: 'home', component: HomePageComponent}
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
