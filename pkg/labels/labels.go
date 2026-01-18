package labels

// Prefixo e labels usados para identificar containers gerenciados pelo OI
const (
	Prefix  = "io.oi."
	Managed = Prefix + "managed"
	Project = Prefix + "project"
	Version = Prefix + "version"
	Domain  = Prefix + "domain"
	Port    = Prefix + "port"
)

// OILabels retorna o conjunto de labels padrão para um container OI
func OILabels(project, version, domain string, port int) map[string]string {
	return map[string]string{
		Managed: "true",
		Project: project,
		Version: version,
		Domain:  domain,
		Port:    itoa(port),
	}
}

// ManagedFilter retorna o filtro para listar containers gerenciados
func ManagedFilter() string {
	return Managed + "=true"
}

// ProjectFilter retorna o filtro para listar containers de um projeto específico
func ProjectFilter(projectName string) string {
	return Project + "=" + projectName
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [10]byte
	n := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		b[n] = '-'
	}
	return string(b[n:])
}
