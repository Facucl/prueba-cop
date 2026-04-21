Configuración (Devops)

Luego de crear un pipeline lo debemos vincular al grupo de despliegue según el proyecto. 
Los actuales grupos son: 

	- deploy_web
	- deploy_dao
	- deploy_dc
	- deploy_pagos
	- deploy_planes_de_pagos
	- deploy_impuestos
	- deploy_sellos
	- deploy_dif
	- deploy_fta
	- deploy_copernico

Para asignar un pipeline a un grupo. Debemos acceder al Repositorio Helm-Chart y en el apartado "Openshift-cluster-auth" podemos observar que están los archivos de Values por Cluster: 
 

 https://git.rentasweb.gob.ar/DevOps/helm-charts/tree/master/charts/openshift-cluster-auth

Dentro de cada archivo Values podemos encontrar los grupos y las aplicaciones asignadas a éstos. 

	1- Identificar el grupo correspondiente al proyecto y dentro del apartado applications, asignar las aplicaciones a las ese grupo tendrá acceso. 
	
	
   
 
	2- En caso que no exista el grupo se debe crear. 


Acceso a Desarrolladores. 

En caso que el desarrollador no tenga acceso al pipeline, debe solicitar que se lo asigne al grupo asignado a ese pipeline. (Verificar que el proyecto este asignado a el un grupo de deploy) Pedir vía solicitud GLPI accesos a los usuarios al grupo correspondiente. 
