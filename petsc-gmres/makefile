include ${PETSC_DIR}/lib/petsc/conf/variables
include ${PETSC_DIR}/lib/petsc/conf/rules

all: GMRES.o  chkopts
	${CLINKER} -o GMRES GMRES.o  ${PETSC_KSP_LIB}

run: all
	${MPIEXEC} ./GMRES -if ${IFPATH} -of ${OFPATH} -imax ${IMAX}
	
cleanall:
	${RM} GMRES.o GMRES
