static char help[] = "Use GMRES to solve linear systems\n\n";

#include <petscksp.h>
#include <petscbag.h>


typedef struct{
    PetscInt       i,imax;
    char           ifname[PETSC_MAX_PATH_LEN],ofname[PETSC_MAX_PATH_LEN];
} Parameter;

int main(int argc,char **argv) {
    PetscViewer    ifd,ofd;                   //file viewer
    Vec            b;                         //RHS and approx solution (sharing memory)
    Mat            A;                         //linear system matrix
    KSP            ksp;                       //linear solver context
    PC             pc;                        //PC context
    PetscErrorCode ierr;
    Parameter      *params;
    PetscBag       bag;

    ierr = PetscInitialize(&argc,&argv,(char*)0,help);CHKERRQ(ierr);    

    ierr = PetscBagCreate(PETSC_COMM_WORLD,sizeof(Parameter),&bag);CHKERRQ(ierr);
    ierr = PetscBagGetData(bag,(void**)&params);CHKERRQ(ierr);

    ierr = PetscBagSetName(bag,"ParameterBag","contains parameters for script");CHKERRQ(ierr);
    ierr = PetscBagRegisterString(bag,&params->ifname,PETSC_MAX_PATH_LEN,"Ab.ptsc","if","Name of input file file");CHKERRQ(ierr);
    ierr = PetscBagRegisterString(bag,&params->ofname,PETSC_MAX_PATH_LEN,"sol.matlab","of","Name of output file file");CHKERRQ(ierr);
    ierr = PetscBagRegisterInt   (bag,&params->imax, 0,"imax","Number of vectors");CHKERRQ(ierr);

    // Open input file
    ierr = PetscViewerBinaryOpen(PETSC_COMM_WORLD,params->ifname,FILE_MODE_READ,&ifd);CHKERRQ(ierr);
    // Open output file
    ierr = PetscViewerASCIIOpen(PETSC_COMM_WORLD,params->ofname,&ofd); CHKERRQ(ierr);
    ierr = PetscViewerPushFormat(ofd,PETSC_VIEWER_ASCII_MATLAB); CHKERRQ(ierr);

    // Load the matrix.
    ierr = MatCreate(PETSC_COMM_WORLD,&A);CHKERRQ(ierr);
    ierr = MatLoad(A,ifd);CHKERRQ(ierr);

    // Solver options and tolerances.
    ierr = KSPCreate(PETSC_COMM_WORLD,&ksp);CHKERRQ(ierr);
    ierr = KSPSetType(ksp,KSPGMRES);CHKERRQ(ierr);//by default..
    ierr = KSPSetOperators(ksp,A,A);CHKERRQ(ierr);
    ierr = KSPSetTolerances(ksp,1e-8,1e-16,1e4,500);CHKERRQ(ierr);

    ierr = KSPGetPC(ksp,&pc);CHKERRQ(ierr);
    ierr = PCSetType(pc,PCSOR);CHKERRQ(ierr);

    ierr = KSPSetFromOptions(ksp);CHKERRQ(ierr);

    ierr = VecCreate(PETSC_COMM_WORLD,&b);CHKERRQ(ierr);
    for (params->i=0; params->i < params->imax; params->i++){
        ierr = VecLoad(b,ifd); CHKERRQ(ierr);
        ierr = KSPSolve(ksp,b,b);CHKERRQ(ierr);
        ierr = VecView(b,ofd);CHKERRQ(ierr);
    }

    //Free work space.
    ierr = PetscViewerDestroy(&ifd);CHKERRQ(ierr);
    ierr = PetscViewerPopFormat(ofd); CHKERRQ(ierr);
    ierr = PetscViewerDestroy(&ofd);CHKERRQ(ierr);
    ierr = KSPDestroy(&ksp);CHKERRQ(ierr);
    ierr = VecDestroy(&b);CHKERRQ(ierr);
    ierr = MatDestroy(&A);CHKERRQ(ierr);
    ierr = PetscBagDestroy(&bag);CHKERRQ(ierr);
    ierr = PetscFinalize();
    return 0;
}
