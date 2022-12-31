package main

import (
	"log"
	"net/http"
	"os"

	module "imuslab.com/arozos/mod/modules"
	prout "imuslab.com/arozos/mod/prouter"
	"imuslab.com/arozos/mod/utils"
)

var (
	moduleHandler *module.ModuleHandler
)

func ModuleServiceInit() {
	//Create a new module handler
	moduleHandler = module.NewModuleHandler(userHandler, *tmp_directory)

	//Register FTP Endpoints
	adminRouter := prout.NewModuleRouter(prout.RouterOption{
		AdminOnly:   true,
		UserHandler: userHandler,
		DeniedHandler: func(w http.ResponseWriter, r *http.Request) {
			errorHandlePermissionDenied(w, r)
		},
	})

	//Pass through the endpoint to authAgent
	http.HandleFunc("/system/modules/list", func(w http.ResponseWriter, r *http.Request) {
		authAgent.HandleCheckAuth(w, r, moduleHandler.ListLoadedModules)
	})
	http.HandleFunc("/system/modules/getDefault", func(w http.ResponseWriter, r *http.Request) {
		authAgent.HandleCheckAuth(w, r, moduleHandler.HandleDefaultLauncher)
	})
	http.HandleFunc("/system/modules/getLaunchPara", func(w http.ResponseWriter, r *http.Request) {
		authAgent.HandleCheckAuth(w, r, moduleHandler.GetLaunchParameter)
	})

	adminRouter.HandleFunc("/system/modules/reload", func(w http.ResponseWriter, r *http.Request) {
		moduleHandler.ReloadAllModules(AGIGateway)
		utils.SendOK(w)
	})

	//Handle module installer. Require admin
	http.HandleFunc("/system/modules/installViaZip", func(w http.ResponseWriter, r *http.Request) {
		//Check if the user is admin
		userinfo, err := userHandler.GetUserInfoFromRequest(w, r)
		if err != nil {
			utils.SendErrorResponse(w, "User not logged in")
			return
		}

		//Validate the user is admin
		if userinfo.IsAdmin() {
			//Get the installation file path
			installerPath, err := utils.PostPara(r, "path")
			if err != nil {
				utils.SendErrorResponse(w, "Invalid installer path")
				return
			}

			fsh, subpath, err := GetFSHandlerSubpathFromVpath(installerPath)
			if err != nil {
				utils.SendErrorResponse(w, "Invalid installer path")
				return
			}

			//Translate it to realpath
			rpath, err := fsh.FileSystemAbstraction.VirtualPathToRealPath(subpath, userinfo.Username)
			if err != nil {
				systemWideLogger.PrintAndLog("Module Installer", "Failed to install module: "+err.Error(), err)
				utils.SendErrorResponse(w, "Invalid installer path")
				return
			}

			//Install it
			moduleHandler.InstallViaZip(rpath, AGIGateway)
		} else {
			//Permission denied
			utils.SendErrorResponse(w, "Permission Denied")
		}

	})

	//Register setting interface for module configuration
	registerSetting(settingModule{
		Name:     "Module List",
		Desc:     "A list of module currently loaded in the system",
		IconPath: "SystemAO/modules/img/small_icon.png",
		Group:    "Module",
		StartDir: "SystemAO/modules/moduleList.html",
	})

	registerSetting(settingModule{
		Name:     "Default Module",
		Desc:     "Default module use to open a file",
		IconPath: "SystemAO/modules/img/small_icon.png",
		Group:    "Module",
		StartDir: "SystemAO/modules/defaultOpener.html",
	})

	if !*disable_subservices {
		registerSetting(settingModule{
			Name:         "Subservices",
			Desc:         "Launch and kill subservices",
			IconPath:     "SystemAO/modules/img/small_icon.png",
			Group:        "Module",
			StartDir:     "SystemAO/modules/subservices.html",
			RequireAdmin: true,
		})
	}

	err := sysdb.NewTable("module")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

}

/*
	Handle endpoint registry for Module installer

*/
func ModuleInstallerInit() {
	//Register module installation setting
	registerSetting(settingModule{
		Name:         "Add & Remove Module",
		Desc:         "Install & Remove Module to the system",
		IconPath:     "SystemAO/modules/img/small_icon.png",
		Group:        "Module",
		StartDir:     "SystemAO/modules/addAndRemove.html",
		RequireAdmin: true,
	})

	//Create new permission router
	router := prout.NewModuleRouter(prout.RouterOption{
		ModuleName:  "System Setting",
		UserHandler: userHandler,
		AdminOnly:   true,
		DeniedHandler: func(w http.ResponseWriter, r *http.Request) {
			errorHandlePermissionDenied(w, r)
		},
	})

	router.HandleFunc("/system/module/install", HandleModuleInstall)

}

//Handle module installation request
func HandleModuleInstall(w http.ResponseWriter, r *http.Request) {
	opr, _ := utils.PostPara(r, "opr")

	if opr == "gitinstall" {
		//Get URL from request
		url, _ := utils.PostPara(r, "url")
		if url == "" {
			utils.SendErrorResponse(w, "Invalid URL")
			return
		}

		//Install the module using git
		err := moduleHandler.InstallModuleViaGit(url, AGIGateway)
		if err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}

		//Reply ok
		utils.SendOK(w)
	} else if opr == "zipinstall" {

	} else if opr == "remove" {
		//Get the module name from list
		module, _ := utils.PostPara(r, "module")
		if module == "" {
			utils.SendErrorResponse(w, "Invalid Module Name")
			return
		}

		//Remove the module
		err := moduleHandler.UninstallModule(module)
		if err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}

		//Reply ok
		utils.SendOK(w)

	} else {
		//List all the modules
		moduleHandler.HandleModuleInstallationListing(w, r)
	}
}
