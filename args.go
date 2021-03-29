package templatedir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v34/github"
	"golang.org/x/oauth2"
)

// Args ...
type Args map[string]interface{}

func (a Args) String() string {
	buf, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Args::String: %s\n", err.Error())
		return ""
	}
	return string(buf)
}

// InitFromOSEnv ...
func (a *Args) InitFromOSEnv() {
	if len(*a) == 0 {
		*a = Args{}
	}
	for _, arg := range os.Environ() {
		parts := strings.SplitN(arg, "=", 2)
		argName := parts[0]
		argValue := parts[1]
		(*a)[argName] = argValue
	}
}

// Author is {{.Author}}
// This repository is named {{.RepoName}}
// Local root of repository is {{.Root}}

// DefaultArgs ...
func DefaultArgs() (Args, error) {

	// curl -s https://api.github.com/users/parro-it
	// curl -s https://api.github.com/repos/parro-it/gomod

	var args Args
	args.InitFromOSEnv()

	info, ok := getRepoInfoFromGHActionEnv()
	if !ok {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		info, err = getRepoInfoFromGit(cwd)
		if err != nil {
			return nil, err
		}
	}

	args["Author"] = info.Author
	args["RepoName"] = info.RepoName
	args["Root"] = info.Root

	user, repo, err := getGHInfos(info)
	if err != nil {
		return nil, err
	}
	args["User"] = user
	args["Repo"] = repo

	return args, nil
}

func getGHInfos(info repoInfo) (*github.User, *github.Repository, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, nil, nil
	}
	ctx, client, err := githubAuth(token)
	if err != nil {
		return nil, nil, err
	}

	user, _, err := client.Users.Get(ctx, info.Author)
	if err != nil {
		return nil, nil, err
	}

	repo, _, err := client.Repositories.Get(ctx, info.Author, info.RepoName)
	if err != nil {
		return nil, nil, err
	}

	return user, repo, nil
}

type repoInfo struct {
	Author   string
	RepoName string
	Root     string
}

func getRepoInfoFromGHActionEnv() (repoInfo, bool) {
	res := repoInfo{}
	ghrepo := os.Getenv("GITHUB_REPOSITORY")
	if ghrepo == "" {
		return res, false
	}
	parts := strings.SplitN(ghrepo, "/", 2)
	res.Author = parts[0]
	res.RepoName = parts[1]
	res.Root = os.Getenv("GITHUB_WORKSPACE")

	return res, true
}

func getRepoInfoFromGit(dir string) (repoInfo, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	res := repoInfo{
		Root: dir,
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return res, fmt.Errorf("getRepoInfoFromGit error: %w\ngit command output:\n%s", err, out)
	}
	outs := strings.TrimRight(string(out), " \t\n\r")
	ghPrefix := "https://github.com/"
	if !strings.HasPrefix(outs, ghPrefix) {
		return res, fmt.Errorf("getRepoInfoFromGit error: unknown provider %s", out)
	}

	outs = strings.TrimPrefix(outs, ghPrefix)

	parts := strings.Split(outs, "/")

	res.Author = parts[0]
	res.RepoName = strings.TrimSuffix(parts[1], ".git")

	return res, nil
}

// githubAuth returns a GitHub client and context.
func githubAuth(token string) (context.Context, *github.Client, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return ctx, client, nil
}

// addRepoSecret will add a secret to a GitHub repo for use in GitHub Actions.
//
// Finally, the secretName and secretValue will determine the name of the secret added and it's corresponding value.
//
// The actual transmission of the secret value to GitHub using the api requires that the secret value is encrypted
// using the public key of the target repo. This encryption must be done using sodium.
//
// First, the public key of the repo is retrieved. The public key comes base64
// encoded, so it must be decoded prior to use in sodiumlib.
//
// Second, the secret value is converted into a slice of bytes.
//
// Third, the secret is encrypted with sodium.CryptoBoxSeal using the repo's decoded public key.
//
// Fourth, the encrypted secret is encoded as a base64 string to be used in a github.EncodedSecret type.
//
// Fifth, The other two properties of the github.EncodedSecret type are determined. The name of the secret to be added
// (string not base64), and the KeyID of the public key used to encrypt the secret.
// This can be retrieved via the public key's GetKeyID method.
//
// Finally, the github.EncodedSecret is passed into the GitHub client.Actions.CreateOrUpdateRepoSecret method to
// populate the secret in the GitHub repo.

/*
func addRepoSecret(ctx context.Context, client *github.Client, owner string, repo, secretName string, secretValue string) error {
	publicKey, _, err := client.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return err
	}

	encryptedSecret, err := encryptSecretWithPublicKey(publicKey, secretName, secretValue)
	if err != nil {
		return err
	}

	if _, err := client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, encryptedSecret); err != nil {
		return fmt.Errorf("Actions.CreateOrUpdateRepoSecret returned error: %v", err)
	}

	return nil
}

func encryptSecretWithPublicKey(publicKey *github.PublicKey, secretName string, secretValue string) (*github.EncryptedSecret, error) {
	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKey.GetKey())
	if err != nil {
		return nil, fmt.Errorf("base64.StdEncoding.DecodeString was unable to decode public key: %v", err)
	}

	secretBytes := []byte(secretValue)
	encryptedBytes, exit := sodium.CryptoBoxSeal(secretBytes, decodedPublicKey)
	if exit != 0 {
		return nil, errors.New("sodium.CryptoBoxSeal exited with non zero exit code")
	}

	encryptedString := base64.StdEncoding.EncodeToString(encryptedBytes)
	keyID := publicKey.GetKeyID()
	encryptedSecret := &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          keyID,
		EncryptedValue: encryptedString,
	}
	return encryptedSecret, nil
}

func _() {
	token := ""
	owner := ""
	repo := ""
	secretName := ""
	secretValue := ""

	ctx, client, err := githubAuth(token)
	if err != nil {
		log.Fatalf("unable to authorize using env GITHUB_AUTH_TOKEN: %v", err)
	}

	if err := addRepoSecret(ctx, client, owner, repo, secretName, secretValue); err != nil {
		log.Fatal(err)
	}

}
*/
